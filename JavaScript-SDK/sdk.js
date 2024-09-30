const axios = require('axios');
const xxhash = require('xxhashjs');
const bls = require('bls.js'); // You might need to find the correct BLS library

// Utility hash function using xxhash
const hash = (input) => xxhash.h64().update(input).digest().toString('hex');

// Function to fetch operators using a GraphQL query
async function getOperators() {
    const subgraphUrl = "https://api.studio.thegraph.com/query/85556/bls_apk_registry/version/latest";
    const query = `
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
    `;
    
    const response = await axios.post(subgraphUrl, { query: query });
    const operators = response.data.data.operators;
    
    operators.forEach((operator) => {
        operator.stake = Math.min(1, parseFloat(operator.stake) / Math.pow(10, 18));
        let publicKeyG2 = `1 ${operator.pubkeyG2_X[1]} ${operator.pubkeyG2_X[0]} ${operator.pubkeyG2_Y[1]} ${operator.pubkeyG2_Y[0]}`;
        operator.public_key_g2 = new bls.G2Point();
        operator.public_key_g2.setStr(publicKeyG2);
    });

    return Object.fromEntries(operators.map(operator => [operator.id, operator]));
}

// Zellular class definition
class Zellular {
    constructor(appName, baseUrl, thresholdPercent = 67) {
        this.appName = appName;
        this.baseUrl = baseUrl;
        this.thresholdPercent = thresholdPercent;
        this.operators = null;
        this.aggregatedPublicKey = new bls.G2Point();

        // Fetch operators and initialize public keys
        getOperators().then(operators => {
            this.operators = operators;
            for (let operator of Object.values(operators)) {
                this.aggregatedPublicKey = this.aggregatedPublicKey.add(operator.public_key_g2);
            }
        });
    }

    // Verify a BLS signature
    verifySignature(message, signatureHex, nonsigners) {
        let totalStake = Object.values(this.operators).reduce((sum, operator) => sum + operator.stake, 0);
        nonsigners = nonsigners.map(id => this.operators[id]);
        let nonsignersStake = nonsigners.reduce((sum, operator) => sum + operator.stake, 0);

        if ((100 * nonsignersStake / totalStake) > (100 - this.thresholdPercent)) {
            return false;
        }

        let publicKey = this.aggregatedPublicKey;
        nonsigners.forEach(operator => {
            publicKey = publicKey.sub(operator.public_key_g2);
        });

        let signature = new bls.Signature();
        signature.setStr(signatureHex);

        message = hash(message);
        return signature.verify(publicKey, message);
    }

    // Verify a finalized batch
    verifyFinalized(data, batchHash, chainingHash) {
        let message = JSON.stringify({
            app_name: this.appName,
            state: 'locked',
            index: data.index,
            hash: batchHash,
            chaining_hash: chainingHash
        });

        let signature = data.finalization_signature;
        let nonsigners = data.nonsigners;
        let result = this.verifySignature(message, signature, nonsigners);

        console.log(`App: ${this.appName}, Index: ${data.index}, Verification Result: ${result}`);
        return result;
    }

    // Fetch finalized batches from the backend
    async getFinalized(after, chainingHash) {
        let res = [];
        let index = chainingHash !== null ? after : after - 1;

        while (true) {
            const resp = await axios.get(`${this.baseUrl}/node/${this.appName}/batches/finalized?after=${index}`);
            const data = resp.data.data;

            if (!data) continue;

            let { batches, finalized } = data;
            if (!chainingHash) {
                chainingHash = data.first_chaining_hash;
                batches = batches.slice(1);
                index += 1;
            }

            for (let batch of batches) {
                index += 1;
                chainingHash = hash(chainingHash + hash(batch));
                res.push(batch);

                if (finalized && index === finalized.index) {
                    const isValid = this.verifyFinalized(finalized, hash(batch), chainingHash);
                    if (!isValid) {
                        throw new Error('Invalid signature');
                    }
                    return { chainingHash, res };
                }
            }
        }
    }

    // Generator to yield batches
    async *batches(after = 0) {
        let chainingHash = after === 0 ? '' : null;

        while (true) {
            let { chainingHash: newChainingHash, res: batches } = await this.getFinalized(after, chainingHash);
            chainingHash = newChainingHash;

            for (let batch of batches) {
                after += 1;
                yield { batch, after };
            }
        }
    }

    // Get the last finalized batch
    async getLastFinalized() {
        const url = `${this.baseUrl}/node/${this.appName}/batches/finalized/last`;
        const resp = await axios.get(url);
        const data = resp.data.data;

        const verified = this.verifyFinalized(data, data.hash, data.chaining_hash);
        if (!verified) {
            throw new Error('Invalid signature');
        }

        return data;
    }

    // Send a batch
    async send(batch, blocking = false) {
        if (blocking) {
            let lastFinalized = await this.getLastFinalized();
            let index = lastFinalized.index;

            for await (let { batch: receivedBatch, after: newIndex } of this.batches(index)) {
                if (batch === JSON.parse(receivedBatch)) {
                    return newIndex;
                }
            }
        }

        const url = `${this.baseUrl}/node/${this.appName}/batches`;
        const resp = await axios.put(url, batch);
        if (resp.status !== 200) {
            throw new Error('Failed to send batch');
        }
    }
}

// Main function to run the example
(async () => {
    const operators = await getOperators();
    const baseUrl = Object.values(operators)[Math.floor(Math.random() * Object.values(operators).length)].socket;

    console.log(baseUrl);

    const verifier = new Zellular('simple_app', baseUrl);
    for await (let { batch, after } of verifier.batches()) {
        const txs = JSON.parse(batch);
        txs.forEach((tx, i) => {
            console.log(after, i, tx);
        });
    }
})();