//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2024 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package common_filters

import (
	"fmt"

	"github.com/weaviate/weaviate/entities/searchparams"
)

const DefaultAlpha = float64(0.75)
const (
	HybridRankedFusion = iota
	HybridRelativeScoreFusion
)
const HybridFusionDefault = HybridRelativeScoreFusion

func ExtractHybridSearch(source map[string]interface{}, explainScore bool) (*searchparams.HybridSearch, error) {
	var subsearches []interface{}
	operandsI := source["operands"]
	if operandsI != nil {
		operands := operandsI.([]interface{})
		for _, operand := range operands {
			operandMap := operand.(map[string]interface{})
			subsearches = append(subsearches, operandMap)
		}
	}
	var args searchparams.HybridSearch
	namedSearchesI := source["searches"]
	if namedSearchesI != nil {
		namedSearchess := namedSearchesI.([]interface{})
		namedSearches := namedSearchess[0].(map[string]interface{})
		// TODO: add bm25 here too
		if namedSearches["nearText"] != nil {
			nearText := namedSearches["nearText"].(map[string]interface{})
			arguments, _ := ExtractNearText(nearText)

			args.NearTextParams = &arguments
		}

		if namedSearches["nearVector"] != nil {
			nearVector := namedSearches["nearVector"].(map[string]interface{})
			arguments, _ := ExtractNearVector(nearVector)
			args.NearVectorParams = &arguments

		}
	}

	var weightedSearchResults []searchparams.WeightedSearchResult

	for _, ss := range subsearches {
		subsearch := ss.(map[string]interface{})
		switch {
		case subsearch["sparseSearch"] != nil:
			bm25 := subsearch["sparseSearch"].(map[string]interface{})
			arguments := ExtractBM25(bm25, explainScore)

			weightedSearchResults = append(weightedSearchResults, searchparams.WeightedSearchResult{
				SearchParams: arguments,
				Weight:       subsearch["weight"].(float64),
				Type:         "bm25",
			})
		case subsearch["nearText"] != nil:
			nearText := subsearch["nearText"].(map[string]interface{})
			arguments, _ := ExtractNearText(nearText)

			weightedSearchResults = append(weightedSearchResults, searchparams.WeightedSearchResult{
				SearchParams: arguments,
				Weight:       subsearch["weight"].(float64),
				Type:         "nearText",
			})

		case subsearch["nearVector"] != nil:
			nearVector := subsearch["nearVector"].(map[string]interface{})
			arguments, _ := ExtractNearVector(nearVector)

			weightedSearchResults = append(weightedSearchResults, searchparams.WeightedSearchResult{
				SearchParams: arguments,
				Weight:       subsearch["weight"].(float64),
				Type:         "nearVector",
			})

		default:
			return nil, fmt.Errorf("unknown subsearch type: %+v", subsearch)
		}
	}

	args.SubSearches = weightedSearchResults

	alpha, ok := source["alpha"]
	if ok {
		args.Alpha = alpha.(float64)
	} else {
		args.Alpha = DefaultAlpha
	}

	if args.Alpha < 0 || args.Alpha > 1 {
		return nil, fmt.Errorf("alpha should be between 0.0 and 1.0")
	}

	query, ok := source["query"]
	if ok {
		args.Query = query.(string)
	}

	fusionType, ok := source["fusionType"]
	if ok {
		args.FusionAlgorithm = fusionType.(int)
	} else {
		args.FusionAlgorithm = HybridFusionDefault
	}
	if _, ok := source["vector"]; ok {
		vector := source["vector"].([]interface{})
		args.Vector = make([]float32, len(vector))
		for i, value := range vector {
			args.Vector[i] = float32(value.(float64))
		}
	}

	if _, ok := source["properties"]; ok {
		properties := source["properties"].([]interface{})
		args.Properties = make([]string, len(properties))
		for i, value := range properties {
			args.Properties[i] = value.(string)
		}
	}

	if _, ok := source["targetVectors"]; ok {
		targetVectors := source["targetVectors"].([]interface{})
		args.TargetVectors = make([]string, len(targetVectors))
		for i, value := range targetVectors {
			args.TargetVectors[i] = value.(string)
		}
	}

	args.Type = "hybrid"

	if args.NearTextParams != nil && args.NearVectorParams != nil {
		return nil, fmt.Errorf("hybrid search cannot have both nearText and nearVector parameters")
	}
	if args.Vector != nil && args.NearTextParams != nil {
		return nil, fmt.Errorf("cannot have both vector and nearTextParams")
	}
	if args.Vector != nil && args.NearVectorParams != nil {
		return nil, fmt.Errorf("cannot have both vector and nearVectorParams")
	}

	return &args, nil
}
