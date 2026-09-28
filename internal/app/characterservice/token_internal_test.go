package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterService_EnsureValidToken(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("do nothing if token is still valid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		token1 := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			RefreshToken: "refresh-old",
		})
		token2 := factory.CreateToken(app.Token{
			AccessToken:   "access-new",
			CharacterID:   character.ID,
			CharacterName: character.EveCharacter.Name,
			RefreshToken:  "refresh-new",
		})
		cs := NewFake(Params{
			Storage: st,
			AuthClient: testutil.AuthClientStub{
				Token: testutil.AuthTokenFromAppToken(token2),
			},
		})
		// when
		changed, err := cs.ensureValidToken(ctx, token1)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, "access-old", token1.AccessToken)
		xassert.Equal(t, "refresh-old", token1.RefreshToken)
	})
	t.Run("should refresh token when expired", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			ExpiresAt:    time.Now().UTC().Add(-10 * time.Second),
			RefreshToken: "refresh-old",
		})
		token2 := factory.CreateToken(app.Token{
			AccessToken:   "access-new",
			CharacterID:   character.ID,
			CharacterName: character.EveCharacter.Name,
			RefreshToken:  "refresh-new",
		})
		cs := NewFake(Params{Storage: st, AuthClient: testutil.AuthClientStub{
			Token: testutil.AuthTokenFromAppToken(token2),
		}})
		// when
		changed, err := cs.ensureValidToken(ctx, token)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, "access-new", token.AccessToken)
		xassert.Equal(t, "refresh-new", token.RefreshToken)
		assert.True(t, token.ExpiresAt.After(time.Now()))
	})
	t.Run("should persist refreshed token when canceled after refresh", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			ExpiresAt:    time.Now().UTC().Add(-10 * time.Second),
			RefreshToken: "refresh-old",
		})
		token2 := factory.CreateToken(app.Token{
			AccessToken:   "access-new",
			CharacterID:   character.ID,
			CharacterName: character.EveCharacter.Name,
			RefreshToken:  "refresh-new",
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cs := NewFake(Params{Storage: st, AuthClient: authClientCancelAfterRefresh{
			AuthClientStub: testutil.AuthClientStub{Token: testutil.AuthTokenFromAppToken(token2)},
			cancel:         cancel,
		}})
		// when
		changed, err := cs.ensureValidToken(ctx, token)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterToken(context.Background(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, "access-new", x.AccessToken)
		xassert.Equal(t, "refresh-new", x.RefreshToken)
	})
}

func TestCharacterService_EnsureValidToken_SingleflightLeaderCanceled(t *testing.T) {
	_, st, factory := testutil.NewDBOnDisk(t)
	character := factory.CreateCharacter()
	factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
		AccessToken:  "access-old",
		CharacterID:  character.ID,
		ExpiresAt:    time.Now().UTC().Add(-10 * time.Second),
		RefreshToken: "refresh-old",
	})
	token2 := factory.CreateToken(app.Token{
		AccessToken:   "access-new",
		CharacterID:   character.ID,
		CharacterName: character.EveCharacter.Name,
		RefreshToken:  "refresh-new",
	})
	ac := authClientBlockRefresh{
		AuthClientStub: testutil.AuthClientStub{Token: testutil.AuthTokenFromAppToken(token2)},
		entered:        make(chan struct{}),
	}
	cs := NewFake(Params{Storage: st, AuthClient: ac})
	getToken := func() *app.CharacterToken {
		x, err := st.GetCharacterToken(context.Background(), character.ID)
		require.NoError(t, err)
		return x
	}

	// leader: cancelable ctx, e.g. tied to the update scheduler
	ctxLeader, cancelLeader := context.WithCancel(context.Background())
	var errLeader error
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		_, errLeader = cs.ensureValidToken(ctxLeader, getToken())
	}()
	select {
	case <-ac.entered:
		// the leader's refresh is now in flight
	case <-time.After(time.Second):
		t.Fatal("leader never reached the refresh")
	}

	// follower: independent, never-canceled ctx, e.g. from a manual reload
	tokenFollower := getToken()
	var errFollower error
	followerDone := make(chan struct{})
	go func() {
		defer close(followerDone)
		_, errFollower = cs.ensureValidToken(context.Background(), tokenFollower)
	}()
	time.Sleep(20 * time.Millisecond) // let the follower join the in-flight singleflight call

	// when
	cancelLeader()
	<-leaderDone
	<-followerDone

	// then
	assert.ErrorIs(t, errLeader, app.ErrCanceled)
	assert.ErrorIs(t, errFollower, app.ErrCanceled, "follower must see the leader's cancel as a cancel")
	xassert.Equal(t, "refresh-old", getToken().RefreshToken)
}

// authClientBlockRefresh blocks the refresh until its ctx is canceled.
type authClientBlockRefresh struct {
	testutil.AuthClientStub
	entered chan struct{}
}

func (s authClientBlockRefresh) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	close(s.entered)
	<-ctx.Done()
	return ctx.Err()
}

// authClientCancelAfterRefresh cancels the ctx after a successful refresh.
type authClientCancelAfterRefresh struct {
	testutil.AuthClientStub
	cancel func()
}

func (s authClientCancelAfterRefresh) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	err := s.AuthClientStub.RefreshToken(ctx, token)
	s.cancel()
	return err
}

func TestTokenSource_New(t *testing.T) {
	token := &app.CharacterToken{
		AccessToken:  "access",
		CharacterID:  42,
		ExpiresAt:    time.Now().Add(20 * time.Minute),
		RefreshToken: "refresh",
	}
	t.Run("should create tokens source", func(t *testing.T) {
		f := func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, nil
		}
		ts := newTokenSource(t.Context(), token, f)
		xassert.Equal(t, token, ts.token)
	})
	t.Run("should panic when trying to create without token", func(t *testing.T) {
		assert.Panics(t, func() {
			newTokenSource(t.Context(), nil, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
				return false, nil
			})
		})
	})
	t.Run("should panic when trying to create without refresher func", func(t *testing.T) {
		assert.Panics(t, func() {
			newTokenSource(t.Context(), token, nil)
		})
	})

}

func TestTokenSource_Token(t *testing.T) {
	t.Run("should return token", func(t *testing.T) {
		// given
		token := &app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  42,
			ExpiresAt:    time.Now().Add(20 * time.Minute),
			RefreshToken: "refresh",
		}
		ts := newTokenSource(t.Context(), token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, nil
		})

		// when
		x, err := ts.Token()

		// then
		require.NoError(t, err)
		xassert.Equal(t, x.AccessToken, token.AccessToken)
		xassert.Equal(t, x.RefreshToken, token.RefreshToken)
		xassert.Equal(t, x.Expiry, token.ExpiresAt)
	})

	t.Run("should return refreshed token when about to expire", func(t *testing.T) {
		// given
		token := &app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  42,
			ExpiresAt:    time.Now().Add(59 * time.Second),
			RefreshToken: "refresh",
		}
		expiresAt2 := time.Now().Add(20 * time.Minute)
		refresher := func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			ct.AccessToken = "access2"
			ct.RefreshToken = "refresh2"
			ct.ExpiresAt = expiresAt2
			return true, nil
		}
		ts := newTokenSource(t.Context(), token, refresher)

		// when
		x, err := ts.Token()

		// then
		require.NoError(t, err)
		xassert.Equal(t, x.AccessToken, "access2")
		xassert.Equal(t, x.RefreshToken, "refresh2")
		xassert.Equal(t, x.Expiry, expiresAt2)
	})

	t.Run("should return error when refresh failed", func(t *testing.T) {
		// given
		token := &app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  42,
			ExpiresAt:    time.Now().Add(-1 * time.Minute),
			RefreshToken: "refresh",
		}
		ts := newTokenSource(t.Context(), token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, fmt.Errorf("some error")
		})

		// when
		_, err := ts.Token()

		// then
		require.Error(t, err)
	})
}
