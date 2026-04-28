package rag

import (
	core "dappco.re/go"
	"github.com/qdrant/go-client/qdrant"
)

type ax7QdrantAPI struct {
	closeCalled bool
	healthErr   error
	closeErr    error

	collections []string
	listErr     error
	exists      bool
	existsErr   error
	createErr   error
	deleteErr   error
	info        *qdrant.CollectionInfo
	infoErr     error
	upsertErr   error
	queryErr    error

	created *qdrant.CreateCollection
	deleted string
	upsert  *qdrant.UpsertPoints
	query   *qdrant.QueryPoints
	results []*qdrant.ScoredPoint
}

func (f *ax7QdrantAPI) Close() error {
	f.closeCalled = true
	return f.closeErr
}

func (f *ax7QdrantAPI) HealthCheck(core.Context) (*qdrant.HealthCheckReply, error) {
	return &qdrant.HealthCheckReply{Title: "qdrant"}, f.healthErr
}

func (f *ax7QdrantAPI) ListCollections(core.Context) ([]string, error) {
	return f.collections, f.listErr
}

func (f *ax7QdrantAPI) CollectionExists(core.Context, string) (bool, error) {
	return f.exists, f.existsErr
}

func (f *ax7QdrantAPI) CreateCollection(_ core.Context, request *qdrant.CreateCollection) error {
	f.created = request
	return f.createErr
}

func (f *ax7QdrantAPI) DeleteCollection(_ core.Context, name string) error {
	f.deleted = name
	return f.deleteErr
}

func (f *ax7QdrantAPI) GetCollectionInfo(core.Context, string) (*qdrant.CollectionInfo, error) {
	return f.info, f.infoErr
}

func (f *ax7QdrantAPI) Upsert(_ core.Context, request *qdrant.UpsertPoints) (*qdrant.UpdateResult, error) {
	f.upsert = request
	return &qdrant.UpdateResult{}, f.upsertErr
}

func (f *ax7QdrantAPI) Query(_ core.Context, request *qdrant.QueryPoints) ([]*qdrant.ScoredPoint, error) {
	f.query = request
	return f.results, f.queryErr
}

func ax7CollectionInfo(points uint64, size uint64, status qdrant.CollectionStatus) *qdrant.CollectionInfo {
	return &qdrant.CollectionInfo{
		Status:      status,
		PointsCount: qdrant.PtrOf(points),
		Config: &qdrant.CollectionConfig{
			Params: &qdrant.CollectionParams{
				VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{Size: size}),
			},
		},
	}
}

func TestAX7_DefaultQdrantConfig_Bad(t *core.T) {
	cfg := DefaultQdrantConfig()

	core.AssertNotEqual(t, "", cfg.Host)
	core.AssertNotEqual(t, 6333, cfg.Port)
}

func TestAX7_DefaultQdrantConfig_Ugly(t *core.T) {
	cfg := DefaultQdrantConfig()
	cfg.Host = "mutated"

	core.AssertEqual(t, "localhost", DefaultQdrantConfig().Host)
	core.AssertEqual(t, "mutated", cfg.Host)
}

func TestAX7_NewQdrantClient_Good(t *core.T) {
	client, err := NewQdrantClient(DefaultQdrantConfig())
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertNoError(t, err)
	core.AssertNotNil(t, client)
}

func TestAX7_NewQdrantClient_Bad(t *core.T) {
	client, err := NewQdrantClient(QdrantConfig{Host: "bad host", Port: -1})

	core.AssertError(t, err)
	core.AssertNil(t, client)
}

func TestAX7_NewQdrantClient_Ugly(t *core.T) {
	client, err := NewQdrantClient(QdrantConfig{Host: "localhost", Port: 6334, APIKey: "token", UseTLS: true})
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertNoError(t, err)
	core.AssertTrue(t, client.config.UseTLS)
}

func TestAX7_NewQdrantStore_Good(t *core.T) {
	client, err := NewQdrantStore("http://localhost:6333")
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertNoError(t, err)
	core.AssertEqual(t, 6334, client.config.Port)
}

func TestAX7_NewQdrantStore_Bad(t *core.T) {
	client, err := NewQdrantStore("http://[::1")

	core.AssertError(t, err)
	core.AssertNil(t, client)
}

func TestAX7_NewQdrantStore_Ugly(t *core.T) {
	client, err := NewQdrantStore("")
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertNoError(t, err)
	core.AssertEqual(t, "localhost", client.config.Host)
}

func TestAX7_QdrantClient_Close_Good(t *core.T) {
	fake := &ax7QdrantAPI{}
	err := (&QdrantClient{client: fake}).Close()

	core.AssertNoError(t, err)
	core.AssertTrue(t, fake.closeCalled)
}

func TestAX7_QdrantClient_Close_Bad(t *core.T) {
	err := (&QdrantClient{}).Close()

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_Close_Ugly(t *core.T) {
	fake := &ax7QdrantAPI{closeErr: core.NewError("close failed")}
	err := (&QdrantClient{client: fake}).Close()

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "close failed")
}

func TestAX7_QdrantClient_HealthCheck_Good(t *core.T) {
	err := (&QdrantClient{client: &ax7QdrantAPI{}}).HealthCheck(core.Background())

	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestAX7_QdrantClient_HealthCheck_Bad(t *core.T) {
	err := (&QdrantClient{}).HealthCheck(core.Background())

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_HealthCheck_Ugly(t *core.T) {
	err := (&QdrantClient{client: &ax7QdrantAPI{healthErr: core.NewError("down")}}).HealthCheck(core.Background())

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "down")
}

func TestAX7_QdrantClient_ListCollections_Good(t *core.T) {
	fake := &ax7QdrantAPI{collections: []string{"alpha", "bravo"}}
	names, err := (&QdrantClient{client: fake}).ListCollections(core.Background())

	core.AssertNoError(t, err)
	core.AssertEqual(t, []string{"alpha", "bravo"}, names)
}

func TestAX7_QdrantClient_ListCollections_Bad(t *core.T) {
	names, err := (&QdrantClient{}).ListCollections(core.Background())

	core.AssertError(t, err)
	core.AssertNil(t, names)
}

func TestAX7_QdrantClient_ListCollections_Ugly(t *core.T) {
	fake := &ax7QdrantAPI{listErr: core.NewError("list failed")}
	names, err := (&QdrantClient{client: fake}).ListCollections(core.Background())

	core.AssertError(t, err)
	core.AssertNil(t, names)
}

func TestAX7_QdrantClient_CollectionExists_Good(t *core.T) {
	exists, err := (&QdrantClient{client: &ax7QdrantAPI{exists: true}}).CollectionExists(core.Background(), "docs")

	core.AssertNoError(t, err)
	core.AssertTrue(t, exists)
}

func TestAX7_QdrantClient_CollectionExists_Bad(t *core.T) {
	exists, err := (&QdrantClient{}).CollectionExists(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertFalse(t, exists)
}

func TestAX7_QdrantClient_CollectionExists_Ugly(t *core.T) {
	exists, err := (&QdrantClient{client: &ax7QdrantAPI{existsErr: core.NewError("exists failed")}}).CollectionExists(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertFalse(t, exists)
}

func TestAX7_QdrantClient_CreateCollection_Good(t *core.T) {
	fake := &ax7QdrantAPI{}
	err := (&QdrantClient{client: fake}).CreateCollection(core.Background(), "docs", 768)

	core.AssertNoError(t, err)
	core.AssertEqual(t, uint64(768), fake.created.GetVectorsConfig().GetParams().GetSize())
}

func TestAX7_QdrantClient_CreateCollection_Bad(t *core.T) {
	err := (&QdrantClient{}).CreateCollection(core.Background(), "docs", 768)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_CreateCollection_Ugly(t *core.T) {
	err := (&QdrantClient{client: &ax7QdrantAPI{createErr: core.NewError("create failed")}}).CreateCollection(core.Background(), "", 0)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "create failed")
}

func TestAX7_QdrantClient_DeleteCollection_Good(t *core.T) {
	fake := &ax7QdrantAPI{}
	err := (&QdrantClient{client: fake}).DeleteCollection(core.Background(), "docs")

	core.AssertNoError(t, err)
	core.AssertEqual(t, "docs", fake.deleted)
}

func TestAX7_QdrantClient_DeleteCollection_Bad(t *core.T) {
	err := (&QdrantClient{}).DeleteCollection(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_DeleteCollection_Ugly(t *core.T) {
	err := (&QdrantClient{client: &ax7QdrantAPI{deleteErr: core.NewError("delete failed")}}).DeleteCollection(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "delete failed")
}

func TestAX7_QdrantClient_CollectionInfo_Good(t *core.T) {
	info, err := (&QdrantClient{client: &ax7QdrantAPI{info: ax7CollectionInfo(3, 768, qdrant.CollectionStatus_Green)}}).CollectionInfo(core.Background(), "docs")

	core.AssertNoError(t, err)
	core.AssertEqual(t, uint64(768), info.VectorSize)
}

func TestAX7_QdrantClient_CollectionInfo_Bad(t *core.T) {
	info, err := (&QdrantClient{}).CollectionInfo(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_QdrantClient_CollectionInfo_Ugly(t *core.T) {
	info, err := (&QdrantClient{client: &ax7QdrantAPI{infoErr: core.NewError("info failed")}}).CollectionInfo(core.Background(), "docs")

	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_QdrantClient_UpsertPoints_Good(t *core.T) {
	fake := &ax7QdrantAPI{}
	err := (&QdrantClient{client: fake}).UpsertPoints(core.Background(), "docs", []Point{{ID: "id", Vector: []float32{0.1}, Payload: map[string]any{"text": "alpha"}}})

	core.AssertNoError(t, err)
	core.AssertEqual(t, "docs", fake.upsert.GetCollectionName())
}

func TestAX7_QdrantClient_UpsertPoints_Bad(t *core.T) {
	err := (&QdrantClient{}).UpsertPoints(core.Background(), "docs", []Point{{ID: "id", Vector: []float32{0.1}}})

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_UpsertPoints_Ugly(t *core.T) {
	err := (&QdrantClient{}).UpsertPoints(core.Background(), "docs", nil)

	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestAX7_QdrantClient_Search_Good(t *core.T) {
	fake := &ax7QdrantAPI{results: []*qdrant.ScoredPoint{{
		Id:    qdrant.NewID("point-1"),
		Score: 0.9,
		Payload: map[string]*qdrant.Value{
			"text":        qdrant.NewValueString("alpha"),
			"source":      qdrant.NewValueString("a.md"),
			"section":     qdrant.NewValueString("Intro"),
			"category":    qdrant.NewValueString("docs"),
			"chunk_index": qdrant.NewValueInt(2),
		},
	}}}
	results, err := (&QdrantClient{client: fake}).Search(core.Background(), "docs", []float32{0.1}, 5, map[string]string{"category": "docs"})

	core.AssertNoError(t, err)
	core.AssertEqual(t, "alpha", results[0].Text)
}

func TestAX7_QdrantClient_Search_Bad(t *core.T) {
	results, err := (&QdrantClient{}).Search(core.Background(), "docs", []float32{0.1}, 5, nil)

	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestAX7_QdrantClient_Search_Ugly(t *core.T) {
	results, err := (&QdrantClient{client: &ax7QdrantAPI{queryErr: core.NewError("query failed")}}).Search(core.Background(), "docs", nil, 0, nil)

	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestAX7_QdrantClient_Add_Good(t *core.T) {
	fake := &ax7QdrantAPI{}
	err := (&QdrantClient{client: fake}).Add(core.Background(), "docs", []Vector{{ID: "id", Values: []float32{0.2}, Payload: map[string]any{"text": "alpha"}}})

	core.AssertNoError(t, err)
	core.AssertEqual(t, "docs", fake.upsert.GetCollectionName())
}

func TestAX7_QdrantClient_Add_Bad(t *core.T) {
	err := (&QdrantClient{}).Add(core.Background(), "docs", []Vector{{ID: "id", Values: []float32{0.2}}})

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not initialized")
}

func TestAX7_QdrantClient_Add_Ugly(t *core.T) {
	err := (&QdrantClient{}).Add(core.Background(), "docs", nil)

	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestAX7_SearchResult_GetText_Good(t *core.T) {
	result := SearchResult{Text: "direct text", Payload: map[string]any{"text": "payload text"}}

	core.AssertEqual(t, "direct text", result.GetText())
	core.AssertNotEmpty(t, result.GetText())
}

func TestAX7_SearchResult_GetText_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetText())
	core.AssertEmpty(t, result.GetText())
}

func TestAX7_SearchResult_GetText_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"text": "payload text"}}

	core.AssertEqual(t, "payload text", result.GetText())
	core.AssertContains(t, result.GetText(), "payload")
}

func TestAX7_SearchResult_GetScore_Good(t *core.T) {
	result := SearchResult{Score: 0.6}

	core.AssertEqual(t, float32(0.6), result.GetScore())
	core.AssertGreater(t, result.GetScore(), float32(0))
}

func TestAX7_SearchResult_GetScore_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, float32(0), result.GetScore())
	core.AssertFalse(t, result.GetScore() > 0)
}

func TestAX7_SearchResult_GetScore_Ugly(t *core.T) {
	result := SearchResult{Score: -0.5}

	core.AssertEqual(t, float32(-0.5), result.GetScore())
	core.AssertLess(t, result.GetScore(), float32(0))
}

func TestAX7_SearchResult_GetSource_Good(t *core.T) {
	result := SearchResult{Source: "direct.md", Payload: map[string]any{"source": "payload.md"}}

	core.AssertEqual(t, "direct.md", result.GetSource())
	core.AssertNotEmpty(t, result.GetSource())
}

func TestAX7_SearchResult_GetSource_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetSource())
	core.AssertEmpty(t, result.GetSource())
}

func TestAX7_SearchResult_GetSource_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"source": "payload.md"}}

	core.AssertEqual(t, "payload.md", result.GetSource())
	core.AssertContains(t, result.GetSource(), "payload")
}

func TestAX7_SearchResult_GetSection_Good(t *core.T) {
	result := SearchResult{Section: "Intro", Payload: map[string]any{"section": "Payload"}}

	core.AssertEqual(t, "Intro", result.GetSection())
	core.AssertNotEmpty(t, result.GetSection())
}

func TestAX7_SearchResult_GetSection_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetSection())
	core.AssertEmpty(t, result.GetSection())
}

func TestAX7_SearchResult_GetSection_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"section": "Payload"}}

	core.AssertEqual(t, "Payload", result.GetSection())
	core.AssertContains(t, result.GetSection(), "Payload")
}

func TestAX7_SearchResult_GetCategory_Good(t *core.T) {
	result := SearchResult{Category: "docs", Payload: map[string]any{"category": "payload"}}

	core.AssertEqual(t, "docs", result.GetCategory())
	core.AssertNotEmpty(t, result.GetCategory())
}

func TestAX7_SearchResult_GetCategory_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetCategory())
	core.AssertEmpty(t, result.GetCategory())
}

func TestAX7_SearchResult_GetCategory_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"category": "payload"}}

	core.AssertEqual(t, "payload", result.GetCategory())
	core.AssertContains(t, result.GetCategory(), "payload")
}

func TestAX7_SearchResult_HasChunkIndex_Good(t *core.T) {
	result := SearchResult{ChunkIndex: 0, ChunkIndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 0, result.GetChunkIndex())
}

func TestAX7_SearchResult_HasChunkIndex_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertFalse(t, result.HasChunkIndex())
	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
}

func TestAX7_SearchResult_HasChunkIndex_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"chunk_index": float64(7)}}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 7, result.GetChunkIndex())
}

func TestAX7_SearchResult_GetChunkIndex_Good(t *core.T) {
	result := SearchResult{ChunkIndex: 5, ChunkIndexPresent: true}

	core.AssertEqual(t, 5, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestAX7_SearchResult_GetChunkIndex_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
	core.AssertFalse(t, result.HasChunkIndex())
}

func TestAX7_SearchResult_GetChunkIndex_Ugly(t *core.T) {
	result := SearchResult{Index: 3, IndexPresent: true, Payload: map[string]any{"chunk_index": 8}}

	core.AssertEqual(t, 3, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}
