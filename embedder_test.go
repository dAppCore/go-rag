package rag

import core "dappco.re/go"

func TestEmbedder_Embedder_Good(t *core.T) {
	var embedder Embedder = newMockEmbedder(3)

	vector, err := embedder.Embed(core.Background(), "hello")
	core.AssertNoError(t, err)
	core.AssertLen(t, vector, 3)
}

func TestEmbedder_Embedder_Bad(t *core.T) {
	var embedder Embedder

	core.AssertNil(t, embedder)
	core.AssertTrue(t, embedder == nil)
}

func TestEmbedder_Embedder_Ugly(t *core.T) {
	embedder := newMockEmbedder(0)
	var _ Embedder = embedder

	vectors, err := embedder.EmbedBatch(core.Background(), []string{"empty-dimension"})
	core.AssertNoError(t, err)
	core.AssertLen(t, vectors, 1)
	core.AssertEmpty(t, vectors[0])
}
