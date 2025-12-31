package eveicon

//go:generate go tool fyne bundle -o resource_gen.go -pkg eveicon resources

//go:generate go run ../../tools/geneveicons/ -p eveicon -out mapping_gen.go

//go:generate go run ../../tools/genschematicids/ -p eveicon -out schematic_gen.go
