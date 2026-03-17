package rag

import (
	"context"
	"fmt"

	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-i18n"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-rag"
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

	// Connect to Qdrant
	qdrantClient, err := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if err != nil {
		return log.E("rag.cmd.collections", "failed to connect to Qdrant", err)
	}
	defer func() { _ = qdrantClient.Close() }()

	// Handle delete
	if deleteCollection != "" {
		exists, err := qdrantClient.CollectionExists(ctx, deleteCollection)
		if err != nil {
			return err
		}
		if !exists {
			return log.E("rag.cmd.collections", fmt.Sprintf("collection not found: %s", deleteCollection), nil)
		}
		if err := qdrantClient.DeleteCollection(ctx, deleteCollection); err != nil {
			return err
		}
		fmt.Printf("Deleted collection: %s\n", deleteCollection)
		return nil
	}

	// List collections
	collections, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		return err
	}

	if len(collections) == 0 {
		fmt.Println("No collections found.")
		return nil
	}

	fmt.Printf("%s\n\n", cli.TitleStyle.Render("Collections"))

	for _, name := range collections {
		if showStats {
			info, err := qdrantClient.CollectionInfo(ctx, name)
			if err != nil {
				fmt.Printf("  %s (error: %v)\n", name, err)
				continue
			}
			fmt.Printf("  %s\n", cli.ValueStyle.Render(name))
			fmt.Printf("    Points:  %d\n", info.PointCount)
			fmt.Printf("    Status:  %s\n", info.Status)
			fmt.Println()
		} else {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}
