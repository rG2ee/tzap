package embedworkflows

import (
	"github.com/tzapio/tzap/internal/logging/tl"
	"github.com/tzapio/tzap/pkg/types"
	"github.com/tzapio/tzap/pkg/tzap"
)

func EmbeddingInspirationWorkflow(query types.QueryRequest, inspirationFiles []string, k int, n int) types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap] {
	return types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap]{
		Name: "embeddingInspirationWorkflow",
		Workflow: func(t *tzap.Tzap) *tzap.Tzap {
			return t.
				ApplyWorkflow(InspirationWorkflow(inspirationFiles)).
				ApplyWorkflow(SearchFilesWorkflow(query, inspirationFiles, k, n))
		},
	}
}

// k is amount of embeddings to be included.
// When using inspiration files, embeddings are likely to be duplicated and as such are filtered out. n is used to increase how many embeddings are fetched but are trimmed to only contain top K after filtering.
func SearchFilesWorkflow(query types.QueryRequest, excludeFiles []string, k int, n int) types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap] {
	return types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap]{
		Name: "searchFilesWorkflow",
		Workflow: func(t *tzap.Tzap) *tzap.Tzap {
			tl.Logger.Println("searchFilesWorkflow")
			if len(query.Queries) == 0 {
				panic("empty embeddings")
			}

			if len(query.Queries) > 1 {
				panic("should only return one embedding")
			}
			embedding := query.Queries[0]
			searchResults, err := t.TG.SearchWithEmbedding(t.C, embedding, n)
			if err != nil {
				panic(err)
			}
			filteredResults := filterSearchResults(searchResults, excludeFiles, k)

			data := types.MappedInterface{
				"searchResults": filteredResults,
			}
			tl.Logger.Println("searchFilesWorkflow ending")
			return t.AddTzap(&tzap.Tzap{Name: "searchResults", Data: data})
		},
	}
}

func filterSearchResults(searchResults types.SearchResults, excludedFiles []string, k int) types.SearchResults {
	filteredResults := []types.Vector{}
	for _, result := range searchResults.Results {
		fileName := result.Metadata["filename"]
		isExcluded := false
		for _, excludedFile := range excludedFiles {
			if fileName == excludedFile {
				isExcluded = true
				break
			}
		}
		if !isExcluded {
			filteredResults = append(filteredResults, result)
		}
		if len(filteredResults) >= k {
			break
		}
	}
	return types.SearchResults{Results: filteredResults}
}