package rag

import core "dappco.re/go"

func TestEmbedder_Embedder_Good(t *core.T) {
	var embedder Embedder = newMockEmbedder(3)

	r := embedder.Embed(core.Background(), "hello")
	vector := r.Value.([]float32)
	core.AssertTrue(t, r.OK)
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

	r := embedder.EmbedBatch(core.Background(), []string{"empty-dimension"})
	vectors := r.Value.([][]float32)
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, vectors, 1)
	core.AssertEmpty(t, vectors[0])
}
