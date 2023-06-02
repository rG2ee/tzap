package applications

import (
	"github.com/tzapio/tzap/cli/cmd/cliworkflows"
	"github.com/tzapio/tzap/pkg/embed"
	"github.com/tzapio/tzap/pkg/embed/localdb/singlewait"
	"github.com/tzapio/tzap/pkg/types"
	"github.com/tzapio/tzap/pkg/tzap"
	"github.com/tzapio/tzap/workflows/code/embedworkflows"
)

type LoadAndSearchEmbeddingArgs struct {
	ExcludeFiles []string `json:"exclude_files"`
	SearchQuery  string   `json:"search_query"`
	K            int      `json:"k"`
	N            int      `json:"n"`
	DisableIndex bool     `json:"disable_index"`
	Yes          bool     `json:"yes"`
}

func LoadAndSearchEmbeddings(args LoadAndSearchEmbeddingArgs) types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap] {
	return types.NamedWorkflow[*tzap.Tzap, *tzap.Tzap]{
		Name: "loadAndSearchEmbeddings",
		Workflow: func(t *tzap.Tzap) *tzap.Tzap {
			queryWait := singlewait.New(func() types.QueryRequest {
				query, err := embed.GetQuery(t, args.SearchQuery)
				if err != nil {
					panic(err)
				}
				return query
			})

			return t.
				ApplyWorkflow(cliworkflows.IndexFilesAndEmbeddings("./", args.DisableIndex, args.Yes)).
				ApplyWorkflow(embedworkflows.EmbeddingInspirationWorkflow(queryWait.GetData(), args.ExcludeFiles, args.K, args.N))
		},
	}
}
