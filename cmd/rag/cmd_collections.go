package rag

import (
	"context"
	"io"

	"dappco.re/go/core"
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
	RunE:  runCollections,
}

func runCollections(cmd *cli.Command, args []string) error {
	ctx := context.Background()
	out := cmd.OutOrStdout()

	// Connect to Qdrant
	qdrantClient, err := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if err != nil {
		return core.E("rag.cmd.collections", "failed to connect to Qdrant", err)
	}
	defer func() { _ = qdrantClient.Close() }()

	// Handle delete
	if deleteCollection != "" {
		exists, err := qdrantClient.CollectionExists(ctx, deleteCollection)
		if err != nil {
			return err
		}
		if !exists {
			return core.E("rag.cmd.collections", core.Sprintf("collection not found: %s", deleteCollection), nil)
		}
		if err := qdrantClient.DeleteCollection(ctx, deleteCollection); err != nil {
			return err
		}
		core.Print(out, "Deleted collection: %s", deleteCollection)
		return nil
	}

	// List collections
	collections, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		return err
	}

	if len(collections) == 0 {
		core.Print(out, "No collections found.")
		return nil
	}

	_, _ = io.WriteString(out, core.Sprintf("%s\n\n", cli.TitleStyle.Render("Collections")))

	for _, name := range collections {
		if showStats {
			info, err := qdrantClient.CollectionInfo(ctx, name)
			if err != nil {
				core.Print(out, "  %s (error: %v)", name, err)
				continue
			}
			core.Print(out, "  %s", cli.ValueStyle.Render(name))
			core.Print(out, "    Points:  %d", info.PointCount)
			core.Print(out, "    Status:  %s", info.Status)
			_, _ = io.WriteString(out, "\n")
		} else {
			core.Print(out, "  %s", name)
		}
	}

	return nil
}
