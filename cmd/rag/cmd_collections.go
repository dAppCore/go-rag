package rag

import (
	"context"
	"io"

	"dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/rag"
)

var (
	listCollections  bool
	showStats        bool
	deleteCollection string
)

var collectionsCmd = &cli.Command{
	Use:   "collections",
	Short: i18n.T("cmd.rag.collections.short"),
	Long:  i18n.T("cmd.rag.collections.long"),
	RunE: func(cmd *cli.Command, args []string) error {
		r := runCollections(cmd, args)
		if r.OK {
			return nil
		}
		if err, ok := r.Value.(error); ok {
			return err
		}
		return core.NewError(r.Error())
	},
}

// runCollections handles list, stats, and delete collection operations.
func runCollections(cmd *cli.Command, args []string) core.Result {
	ctx := context.Background()
	out := cmd.OutOrStdout()

	// Connect to Qdrant
	qdrantResult := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if !qdrantResult.OK {
		return core.Fail(core.E("rag.cmd.collections", "failed to connect to Qdrant", core.NewError(qdrantResult.Error())))
	}
	qdrantClient := qdrantResult.Value.(*rag.QdrantClient)
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Print(out, "Qdrant close failed: %s", r.Error())
		}
	}()

	// Handle delete
	if deleteCollection != "" {
		existsResult := qdrantClient.CollectionExists(ctx, deleteCollection)
		if !existsResult.OK {
			return existsResult
		}
		exists := existsResult.Value.(bool)
		if !exists {
			return core.Fail(core.E("rag.cmd.collections", core.Sprintf("collection not found: %s", deleteCollection), nil))
		}
		if r := qdrantClient.DeleteCollection(ctx, deleteCollection); !r.OK {
			return r
		}
		core.Print(out, "Deleted collection: %s", deleteCollection)
		return core.Ok(deleteCollection)
	}

	// List collections
	collectionsResult := qdrantClient.ListCollections(ctx)
	if !collectionsResult.OK {
		return collectionsResult
	}
	collections := collectionsResult.Value.([]string)

	if len(collections) == 0 {
		core.Print(out, "No collections found.")
		return core.Ok(collections)
	}

	_, _ = io.WriteString(out, core.Sprintf("%s\n\n", cli.TitleStyle.Render("Collections")))

	for _, name := range collections {
		if showStats {
			infoResult := qdrantClient.CollectionInfo(ctx, name)
			if !infoResult.OK {
				core.Print(out, "  %s (error: %s)", name, infoResult.Error())
				continue
			}
			info := infoResult.Value.(*rag.CollectionInfo)
			core.Print(out, "  %s", cli.ValueStyle.Render(name))
			core.Print(out, "    Points:  %d", info.PointCount)
			core.Print(out, "    Status:  %s", info.Status)
			_, _ = io.WriteString(out, "\n")
		} else {
			core.Print(out, "  %s", name)
		}
	}

	return core.Ok(collections)
}
