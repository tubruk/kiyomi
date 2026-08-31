package main

import (
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

func main() {
	plug := NewMyAnimeListPlugin()

	sdk.ServePlugin(sdk.ServeOptions{
		Plugin: plug,
		MetadataProviders: map[string]sdk.MetadataProvider{
			PluginID: plug,
		},
	})
}
