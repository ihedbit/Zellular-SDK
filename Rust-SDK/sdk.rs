use reqwest::Client;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::time::SystemTime;
use bls12_381::{G2Prepared, G2Affine, pairing};

// Struct for operators
#[derive(Debug, Deserialize)]
struct Operator {
    id: String,
    operator_id: String,
    pubkey_g1_x: Vec<String>,
    pubkey_g1_y: Vec<String>,
    pubkey_g2_x: Vec<String>,
    pubkey_g2_y: Vec<String>,
    socket: String,
    stake: f64,
    public_key_g2: Option<G2Affine>, // Placeholder for G2 affine key
}

// Struct for GraphQL query response
#[derive(Debug, Deserialize)]
struct QueryResponse {
    data: QueryData,
}

#[derive(Debug, Deserialize)]
struct QueryData {
    operators: Vec<Operator>,
}

// Hash function using SHA-256
fn hash(input: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input.as_bytes());
    format!("{:x}", hasher.finalize())
}

// Fetch operators via a GraphQL query
async fn get_operators(client: &Client) -> Result<HashMap<String, Operator>, Box<dyn std::error::Error>> {
    let query = r#"
        query MyQuery {
          operators {
            id
            operatorId
            pubkeyG1_X
            pubkeyG1_Y
            pubkeyG2_X
            pubkeyG2_Y
            socket
            stake
          }
        }
    "#;

    let subgraph_url = "https://api.studio.thegraph.com/query/85556/bls_apk_registry/version/latest";

    let resp = client
        .post(subgraph_url)
        .json(&serde_json::json!({ "query": query }))
        .send()
        .await?;

    let response_json: QueryResponse = resp.json().await?;
    let mut operators = HashMap::new();

    for mut operator in response_json.data.operators {
        operator.stake = f64::min(1.0, operator.stake / 10f64.powi(18));

        // Here, we should set the G2 key (use a proper BLS library for Rust)
        let public_key_g2 = G2Affine::identity(); // Placeholder
        operator.public_key_g2 = Some(public_key_g2);

        operators.insert(operator.id.clone(), operator);
    }

    Ok(operators)
}

// Struct for Zellular
struct Zellular {
    app_name: String,
    base_url: String,
    threshold_percent: f64,
    operators: HashMap<String, Operator>,
    aggregated_public_key: G2Affine,
}

impl Zellular {
    pub async fn new(app_name: &str, base_url: &str, threshold_percent: f64) -> Result<Self, Box<dyn std::error::Error>> {
        let client = Client::new();
        let operators = get_operators(&client).await?;

        // Aggregate G2 public keys
        let mut aggregated_public_key = G2Affine::identity();
        for operator in operators.values() {
            aggregated_public_key = aggregated_public_key + operator.public_key_g2.unwrap();
        }

        Ok(Self {
            app_name: app_name.to_string(),
            base_url: base_url.to_string(),
            threshold_percent,
            operators,
            aggregated_public_key,
        })
    }

    // Verify a BLS signature (placeholder, adjust with a real BLS library)
    pub fn verify_signature(&self, message: &str, signature_hex: &str, nonsigners: Vec<String>) -> bool {
        let total_stake: f64 = self.operators.values().map(|op| op.stake).sum();
        let nonsigners_stake: f64 = nonsigners.iter().map(|id| self.operators.get(id).unwrap().stake).sum();

        if (100.0 * nonsigners_stake / total_stake) > (100.0 - self.threshold_percent) {
            return false;
        }

        let mut public_key = self.aggregated_public_key;
        for nonsigner in nonsigners {
            public_key = public_key - self.operators.get(&nonsigner).unwrap().public_key_g2.unwrap();
        }

        // Replace with BLS signature decoding and verification
        let message_hash = hash(message);
        let signature = G2Affine::identity(); // Placeholder
        pairing(&G2Prepared::from(public_key), &signature).is_zero() // Adjust this line with the actual BLS verification logic
    }

    // Fetch finalized batches (simplified version)
    pub async fn get_finalized(&self, after: i32, chaining_hash: Option<String>) -> Result<(String, Vec<String>), Box<dyn std::error::Error>> {
        let client = Client::new();
        let mut res = Vec::new();
        let mut index = if chaining_hash.is_some() { after } else { after - 1 };

        loop {
            let url = format!("{}/node/{}/batches/finalized?after={}", self.base_url, self.app_name, index);
            let resp = client.get(&url).send().await?.json::<serde_json::Value>().await?;

            if resp["data"].is_null() {
                continue;
            }

            let batches = resp["data"]["batches"].as_array().unwrap_or(&vec![]);
            let finalized = &resp["data"]["finalized"];

            for batch in batches {
                index += 1;
                let chaining_hash = chaining_hash.clone().unwrap_or_else(|| "".to_string()) + &hash(batch.as_str().unwrap());
                res.push(batch.as_str().unwrap().to_string());

                if finalized != &serde_json::Value::Null && index == finalized["index"].as_i64().unwrap() as i32 {
                    return Ok((chaining_hash, res));
                }
            }
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::new();
    let operators = get_operators(&client).await?;
    let base_url = operators.values().next().unwrap().socket.clone();

    println!("Base URL: {}", base_url);

    let verifier = Zellular::new("simple_app", &base_url, 67.0).await?;
    let (chaining_hash, batches) = verifier.get_finalized(0, None).await?;

    for (i, batch) in batches.iter().enumerate() {
        println!("Batch {}: {}", i + 1, batch);
    }

    Ok(())
}