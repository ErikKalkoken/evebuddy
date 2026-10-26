package characterservice

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
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
}

func TestCharacterService_EnsureValidToken_RefreshTimeout(t *testing.T) {
	// given
	_, st, factory := testutil.NewDBOnDisk(t)
	synctest.Test(t, func(t *testing.T) {
		character := factory.CreateCharacter()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			ExpiresAt:    time.Now().UTC().Add(-10 * time.Second),
			RefreshToken: "refresh-old",
		})
		cs := NewFake(Params{Storage: st, AuthClient: authClientWaitForCtx{}})
		start := time.Now()
		// when
		_, err := cs.ensureValidToken(context.Background(), token)
		// then
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NotErrorIs(t, err, app.ErrCanceled)
		xassert.Equal(t, tokenRefreshTimeout, time.Since(start))
		x, err := st.GetCharacterToken(context.Background(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, "refresh-old", x.RefreshToken)
	})
}

func TestCharacterService_EnsureValidToken_SingleflightLeaderCanceled(t *testing.T) {
	// given
	_, st, factory := testutil.NewDBOnDisk(t)
	synctest.Test(t, func(t *testing.T) {
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
		getToken := func() *app.CharacterToken {
			x, err := st.GetCharacterToken(context.Background(), character.ID)
			require.NoError(t, err)
			return x
		}
		ac := authClientBlockRefresh{
			AuthClientStub: testutil.AuthClientStub{Token: testutil.AuthTokenFromAppToken(token2)},
			release:        make(chan struct{}),
		}
		cs := NewFake(Params{Storage: st, AuthClient: ac})

		// leader: cancelable ctx, e.g. from a game search
		ctxLeader, cancelLeader := context.WithCancel(context.Background())
		tokenLeader := getToken()
		var errLeader error
		leaderDone := make(chan struct{})
		go func() {
			defer close(leaderDone)
			_, errLeader = cs.ensureValidToken(ctxLeader, tokenLeader)
		}()
		synctest.Wait() // the leader's refresh is now in flight

		// follower: independent, never-canceled ctx, e.g. from a section update
		tokenFollower := getToken()
		var errFollower error
		followerDone := make(chan struct{})
		go func() {
			defer close(followerDone)
			_, errFollower = cs.ensureValidToken(context.Background(), tokenFollower)
		}()
		synctest.Wait() // the follower has joined the in-flight singleflight call

		// when
		cancelLeader()
		synctest.Wait()

		// then
		select {
		case <-leaderDone:
		default:
			assert.Fail(t, "leader waited for the refresh after cancel")
		}
		close(ac.release)
		<-leaderDone
		<-followerDone
		assert.ErrorIs(t, errLeader, app.ErrCanceled)
		require.NoError(t, errFollower)
		xassert.Equal(t, "refresh-new", tokenFollower.RefreshToken)
		xassert.Equal(t, "refresh-new", getToken().RefreshToken)
	})
}

func TestCharacterService_EnsureValidToken_Stop(t *testing.T) {
	t.Run("should wait for a refreshed token to be persisted", func(t *testing.T) {
		// given
		_, st, factory := testutil.NewDBOnDisk(t)
		synctest.Test(t, func(t *testing.T) {
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
			ac := authClientBlockRefreshIgnoreCtx{
				AuthClientStub: testutil.AuthClientStub{Token: testutil.AuthTokenFromAppToken(token2)},
				release:        make(chan struct{}),
			}
			cs := NewFake(Params{Storage: st, AuthClient: ac})
			ctx, cancel := context.WithCancel(context.Background())
			callerDone := make(chan struct{})
			go func() {
				defer close(callerDone)
				_, _ = cs.ensureValidToken(ctx, token)
			}()
			synctest.Wait() // the refresh is now in flight
			cancel()
			<-callerDone
			// when
			stopDone := make(chan struct{})
			go func() {
				defer close(stopDone)
				cs.Stop()
			}()
			synctest.Wait()
			// then
			select {
			case <-stopDone:
				assert.Fail(t, "Stop returned before the refresh finished")
			default:
			}
			close(ac.release)
			<-stopDone
			x, err := st.GetCharacterToken(context.Background(), character.ID)
			require.NoError(t, err)
			xassert.Equal(t, "refresh-new", x.RefreshToken)
		})
	})
	t.Run("should not start a refresh after stop", func(t *testing.T) {
		// given
		_, st, factory := testutil.NewDBOnDisk(t)
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
		cs := NewFake(Params{Storage: st, AuthClient: testutil.AuthClientStub{Token: testutil.AuthTokenFromAppToken(token2)}})
		cs.Stop()
		// when
		_, err := cs.ensureValidToken(context.Background(), token)
		// then
		assert.ErrorIs(t, err, app.ErrCanceled)
		x, err := st.GetCharacterToken(context.Background(), character.ID)
		require.NoError(t, err)
		xassert.Equal(t, "refresh-old", x.RefreshToken)
	})
}

// authClientBlockRefreshIgnoreCtx blocks the refresh until released
// and then succeeds, like a response that arrived before a cancel.
type authClientBlockRefreshIgnoreCtx struct {
	testutil.AuthClientStub
	release chan struct{}
}

func (s authClientBlockRefreshIgnoreCtx) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	<-s.release
	return s.AuthClientStub.RefreshToken(ctx, token)
}

// authClientBlockRefresh blocks the refresh until released
// and fails if its ctx was canceled meanwhile.
type authClientBlockRefresh struct {
	testutil.AuthClientStub
	release chan struct{}
}

func (s authClientBlockRefresh) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	<-s.release
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.AuthClientStub.RefreshToken(ctx, token)
}

// authClientWaitForCtx blocks the refresh until its ctx is done.
type authClientWaitForCtx struct {
	testutil.AuthClientStub
}

func (authClientWaitForCtx) RefreshToken(ctx context.Context, _ *eveauth.Token) error {
	<-ctx.Done()
	return ctx.Err()
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
