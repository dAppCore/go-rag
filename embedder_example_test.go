package rag

import core "dappco.re/go"

func ExampleEmbedder() {
	var embedder Embedder = newMockEmbedder(2)
	vector, err := embedder.Embed(core.Background(), "hello")
	core.Println(err == nil, len(vector))
	// Output: true 2
}
