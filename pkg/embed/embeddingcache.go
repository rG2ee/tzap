package embed

import (
	"encoding/json"
	"os"

	"github.com/tzapio/tzap/internal/logging/tl"
	"github.com/tzapio/tzap/pkg/embed/localdb"
	"github.com/tzapio/tzap/pkg/types"
	"github.com/tzapio/tzap/pkg/tzap"
	"github.com/tzapio/tzap/pkg/util/reflectutil"
)

type EmbeddingCache struct {
	embeddingCacheDB  *localdb.FileDB[string]
	filesTimestampsDB *localdb.FileDB[int64]
}

func NewEmbeddingCache(filesTimestampsDB *localdb.FileDB[int64]) *EmbeddingCache {
	embeddingCacheDB, err := localdb.NewFileDB[string]("./.tzap-data/embeddingsCache.db")
	if err != nil {
		panic(err)
	}
	return &EmbeddingCache{embeddingCacheDB, filesTimestampsDB}
}
func (ec *EmbeddingCache) GetCachedEmbeddings(embeddings types.Embeddings) types.Embeddings {
	var storedFiles map[string]struct{} = map[string]struct{}{}

	tl.Logger.Println("Getting cached embeddings", len(embeddings.Vectors))
	var cachedEmbeddings []types.Vector

	for _, vector := range embeddings.Vectors {
		splitPart := vector.Metadata.SplitPart
		kv, exists := ec.embeddingCacheDB.ScanGet(splitPart)
		if exists {
			if !reflectutil.IsZero(kv.Value) {
				var float32Vector = [1536]float32(make([]float32, 1536))
				json.Unmarshal([]byte(kv.Value), &float32Vector)
				if len(float32Vector) == 1536 {
					vector := types.Vector{
						ID:        vector.ID,
						TimeStamp: 0,
						Metadata:  vector.Metadata,
						Values:    float32Vector,
					}

					cachedEmbeddings = append(cachedEmbeddings, vector)
					storedFiles[vector.Metadata.Filename] = struct{}{}
					continue
				} else {
					println("invalid vector length", splitPart)
					continue
				}
			}
		}
		println("Warning: %s is uncached.", vector.ID)
	}
	if len(storedFiles) > 0 {
		var keyvals []types.KeyValue[int64]
		for file := range storedFiles {
			fileStat, err := os.Stat(file)
			if err != nil {
				panic(err)
			}
			keyvals = append(keyvals, types.KeyValue[int64]{Key: file, Value: fileStat.ModTime().UnixNano()})
		}
		added, err := ec.filesTimestampsDB.BatchSet(keyvals)
		if err != nil {
			panic("failing to store changed files should not happend and has probably caused some kind of corruption")
		}
		tl.Logger.Printf("Added %d files to md5 cache. Total: %d", added, len(storedFiles))
	}
	return types.Embeddings{Vectors: cachedEmbeddings}
}

func (ec *EmbeddingCache) GetUncachedEmbeddings(embeddings types.Embeddings) types.Embeddings {
	var uncachedEmbeddings []types.Vector

	for _, vector := range embeddings.Vectors {
		splitPart := vector.Metadata.SplitPart
		kv, exists := ec.embeddingCacheDB.ScanGet(splitPart)
		if !exists || reflectutil.IsZero(kv.Value) {
			uncachedEmbeddings = append(uncachedEmbeddings, vector)
		}

	}
	return types.Embeddings{Vectors: uncachedEmbeddings}
}

func (ec *EmbeddingCache) FetchThenCacheNewEmbeddings(t *tzap.Tzap, uncachedEmbeddings types.Embeddings) error {
	var storedFiles map[string]struct{} = map[string]struct{}{}

	if len(uncachedEmbeddings.Vectors) > 0 {
		batchSize := 100

		for i := 0; i < len(uncachedEmbeddings.Vectors); i += batchSize {
			end := i + batchSize
			if end > len(uncachedEmbeddings.Vectors) {
				end = len(uncachedEmbeddings.Vectors)
			}

			batch := uncachedEmbeddings.Vectors[i:end]
			var inputStrings []string
			for _, vector := range batch {
				storedFiles[vector.Metadata.Filename] = struct{}{}
				inputStrings = append(inputStrings, vector.Metadata.SplitPart)
			}

			embeddingsResult, err := t.TG.FetchEmbedding(t.C, inputStrings...)
			if err != nil {
				return err
			}

			cacheKeyVal := []types.KeyValue[string]{}
			for i, embedding := range embeddingsResult {
				embBytes, err := json.Marshal(embedding)
				if err != nil {
					return err
				}
				cacheKeyVal = append(cacheKeyVal, types.KeyValue[string]{Key: inputStrings[i], Value: string(embBytes)})
			}

			added, err := ec.embeddingCacheDB.BatchSet(cacheKeyVal)
			if err != nil {
				return err
			}
			tl.UILogger.Println("Added", added, "embeddings to cache")
		}
		if len(storedFiles) > 0 {
			var keyvals []types.KeyValue[int64]
			for file := range storedFiles {
				fileStat, err := os.Stat(file)
				if err != nil {
					return err
				}
				keyvals = append(keyvals, types.KeyValue[int64]{Key: file, Value: fileStat.ModTime().UnixNano()})
			}
			added, err := ec.filesTimestampsDB.BatchSet(keyvals)
			if err != nil {
				panic("failing to store changed files should not happend and has probably caused some kind of corruption")
			}
			tl.Logger.Printf("Added %d files to md5 cache", added)
		}

	}
	return nil
}
