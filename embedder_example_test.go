package rag

import core "dappco.re/go"

func ExampleEmbedder() {
	var embedder Embedder = newMockEmbedder(2)
	r := embedder.Embed(core.Background(), "hello")
	vector := r.Value.([]float32)
	core.Println(r.OK, len(vector))
	// Output: true 2
}
