package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cespare/xxhash"
	"github.com/kilic/bls12-381" // Replace with the correct BLS library
)

// Operator struct holds operator data
type Operator struct {
	ID          string
	OperatorID  string
	PubkeyG1_X  []string
	PubkeyG1_Y  []string
	PubkeyG2_X  []string
	PubkeyG2_Y  []string
	Socket      string
	Stake       float64
	PublicKeyG2 bls12-381.G2Affine
}

// QueryResponse struct holds the GraphQL response data
type QueryResponse struct {
	Data struct {
		Operators []Operator `json:"operators"`
	} `json:"data"`
}

// Hash function using xxhash
func hash(input string) string {
	h := xxhash.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Get operators by making a GraphQL query to the external API
func getOperators() (map[string]Operator, error) {
	subgraphURL := "https://api.studio.thegraph.com/query/85556/bls_apk_registry/version/latest"
	query := `{"query": "query { operators { id operatorId pubkeyG1_X pubkeyG1_Y pubkeyG2_X pubkeyG2_Y socket stake }}"}`

	resp, err := http.Post(subgraphURL, "application/json", bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response QueryResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	operators := make(map[string]Operator)
	for _, operator := range response.Data.Operators {
		operator.Stake = float64(int64(operator.Stake) / (10 ^ 18))

		// Replace this section with actual BLS key handling logic
		publicKeyG2 := bls12-381.G2Affine{} // Adjust with real BLS library methods

		operator.PublicKeyG2 = publicKeyG2
		operators[operator.ID] = operator
	}

	return operators, nil
}

// Zellular struct holds the application and operator information
type Zellular struct {
	AppName            string
	BaseURL            string
	ThresholdPercent   float64
	Operators          map[string]Operator
	AggregatedPublicKey bls12-381.G2Affine
}

// NewZellular initializes a new Zellular instance
func NewZellular(appName, baseURL string, thresholdPercent float64) *Zellular {
	operators, _ := getOperators()
	aggregatedPublicKey := bls12-381.G2Affine{} // Adjust this with real logic to aggregate G2 keys

	// Aggregate all operator public keys
	for _, operator := range operators {
		// Add each operator's G2 public key
		aggregatedPublicKey.Add(&operator.PublicKeyG2)
	}

	return &Zellular{
		AppName:            appName,
		BaseURL:            baseURL,
		ThresholdPercent:   thresholdPercent,
		Operators:          operators,
		AggregatedPublicKey: aggregatedPublicKey,
	}
}

// VerifySignature verifies the BLS signature
func (z *Zellular) VerifySignature(message, signatureHex string, nonsigners []string) bool {
	totalStake := 0.0
	for _, operator := range z.Operators {
		totalStake += operator.Stake
	}

	nonsignersStake := 0.0
	for _, nonsigner := range nonsigners {
		nonsignersStake += z.Operators[nonsigner].Stake
	}

	if 100*nonsignersStake/totalStake > (100 - z.ThresholdPercent) {
		return false
	}

	// Subtract nonsigners' public keys
	publicKey := z.AggregatedPublicKey
	for _, nonsigner := range nonsigners {
		publicKey.Sub(&z.Operators[nonsigner].PublicKeyG2)
	}

	// Decode signature and verify (using real BLS verification)
	messageHash := hash(message)
	signature := bls12-381.Signature{} // Replace this with the actual BLS signature decoding
	return signature.Verify(&publicKey, []byte(messageHash))
}

// GetFinalized retrieves finalized batches from the backend
func (z *Zellular) GetFinalized(after int, chainingHash *string) ([]string, error) {
	var res []string
	index := after
	if chainingHash == nil {
		index = after - 1
	}

	for {
		url := fmt.Sprintf("%s/node/%s/batches/finalized?after=%d", z.BaseURL, z.AppName, index)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		if err != nil || data["data"] == nil {
			continue
		}

		batches := data["data"].(map[string]interface{})["batches"].([]interface{})
		finalized := data["data"].(map[string]interface{})["finalized"].(map[string]interface{})

		for _, batch := range batches {
			batchStr := fmt.Sprintf("%v", batch)
			res = append(res, batchStr)
			index++
			if finalized != nil && index == int(finalized["index"].(float64)) {
				chainingHashStr := chainingHash
				if chainingHash != nil {
					*chainingHash = hash(*chainingHash + hash(batchStr))
				} else {
					chainingHashStr = &batchStr
				}
				return res, nil
			}
		}
	}
}

// Main function demonstrates the Zellular implementation
func main() {
	operators, err := getOperators()
	if err != nil {
		log.Fatalf("Error getting operators: %v", err)
	}
	baseURL := operators[randomOperator(operators)].Socket

	fmt.Println("Base URL:", baseURL)

	verifier := NewZellular("simple_app", baseURL, 67)
	batches, err := verifier.GetFinalized(0, nil)
	if err != nil {
		log.Fatalf("Error getting finalized batches: %v", err)
	}

	for i, batch := range batches {
		fmt.Printf("Batch %d: %s\n", i, batch)
	}
}

// Utility to select a random operator
func randomOperator(operators map[string]Operator) string {
	keys := make([]string, 0, len(operators))
	for key := range operators {
		keys = append(keys, key)
	}
	rand.Seed(time.Now().UnixNano())
	return keys[rand.Intn(len(keys))]
}