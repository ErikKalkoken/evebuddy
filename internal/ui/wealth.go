package ui

import (
	"cmp"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/ui/charts"
)

type wealthArea struct {
	content fyne.CanvasObject
	charts  *fyne.Container
	top     *widget.Label
	ui      *ui
}

func (u *ui) newWealthArea() *wealthArea {
	a := &wealthArea{
		ui:     u,
		charts: container.NewVBox(),
		top:    widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.content = container.NewBorder(
		container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil,
		container.NewScroll(a.charts))
	return a
}

type dataRow struct {
	label  string
	wallet float64
	assets float64
	total  float64
}

func (a *wealthArea) refresh() {
	data, err := a.compileData()
	if err != nil {
		slog.Error("Failed to fetch data for charts", "err", err)
		a.top.Text = fmt.Sprintf("Failed to fetch data for charts: %s", humanize.Error(err))
		a.top.Importance = widget.DangerImportance
		a.top.Refresh()
		return
	}
	hasChanged, err := hasDataChanged(a.ui.sv.Dictionary, data)
	if err != nil {
		panic(err)
	}
	if !hasChanged {
		return
	}
	cb := charts.NewChartBuilder()

	charactersData := make([]charts.Value, len(data))
	for i, r := range data {
		charactersData[i] = charts.Value{Label: r.label, Value: r.assets + r.wallet}
	}
	charactersChart := cb.Render(charts.Pie, "Total Wealth By Character", charactersData)

	var totalWallet, totalAssets float64
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	typesData := make([]charts.Value, 2)
	typesData[0] = charts.Value{Label: "Assets", Value: totalAssets}
	typesData[1] = charts.Value{Label: "Wallet", Value: totalWallet}
	typesChart := cb.Render(charts.Pie, "Total Wealth By Type", typesData)

	assetsData := make([]charts.Value, len(data))
	for i, r := range data {
		assetsData[i] = charts.Value{Label: r.label, Value: r.assets}
	}
	assetsChart := cb.Render(charts.Bar, "Assets Value By Character", assetsData)

	walletData := make([]charts.Value, len(data))
	for i, r := range data {
		walletData[i] = charts.Value{Label: r.label, Value: r.wallet}
	}
	walletChart := cb.Render(charts.Bar, "Wallet Balance By Character", walletData)

	a.charts.RemoveAll()
	a.charts.Add(container.NewHBox(charactersChart, typesChart))
	a.charts.Add(assetsChart)
	a.charts.Add(walletChart)

	var total float64
	for _, r := range data {
		total += r.assets + r.wallet
	}
	a.top.SetText(fmt.Sprintf("Total: %s", humanize.Number(total, 1)))
}

func (a *wealthArea) compileData() ([]dataRow, error) {
	cc, err := a.ui.sv.Characters.ListCharacters(context.TODO())
	if err != nil {
		return nil, err
	}
	data := make([]dataRow, 0)
	for _, c := range cc {
		wallet := c.WalletBalance.Float64
		x, err := a.ui.sv.Characters.CharacterAssetTotalValue(c.ID)
		if err != nil {
			return nil, err
		}
		assets := x.Float64
		label := c.EveCharacter.Name
		r := dataRow{
			label:  label,
			assets: assets,
			wallet: wallet,
			total:  assets + wallet,
		}
		data = append(data, r)
	}
	slices.SortFunc(data, func(a, b dataRow) int {
		return cmp.Compare(a.total, b.total) * -1
	})
	return data, nil
}

func hasDataChanged(dt *dictionary.DictionaryService, data any) (bool, error) {
	hash, err := calcContentHash(data)
	if err != nil {
		return false, err
	}
	key := "wealth-data-hash"
	oldHash, ok, err := dt.String(key)
	if err != nil {
		return false, err
	}
	if ok && oldHash == hash {
		return false, nil
	}
	if err := dt.SetString(key, hash); err != nil {
		return false, err
	}
	return true, nil
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
