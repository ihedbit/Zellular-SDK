import json
import time
import random
from typing import Any

import requests
from eigensdk.crypto.bls import attestation
from eigensdk.chainio.clients.builder import BuildAllConfig, build_all

import xxhash

hash = xxhash.xxh128_hexdigest


def get_operators():
    subgraph_url = (
        "https://api.studio.thegraph.com/query/85556/bls_apk_registry/version/latest"
    )
    query = """
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
    """
    resp = requests.post(subgraph_url, json={"query": query})
    operators = resp.json()["data"]["operators"]

    for operator in operators:
        operator["stake"] = min(1, float(operator["stake"]) / (10**18))
        public_key_g2 = (
            "1 "
            + operator["pubkeyG2_X"][1]
            + " "
            + operator["pubkeyG2_X"][0]
            + " "
            + operator["pubkeyG2_Y"][1]
            + " "
            + operator["pubkeyG2_Y"][0]
        )
        operator["public_key_g2"] = attestation.new_zero_g2_point()
        operator["public_key_g2"].setStr(public_key_g2.encode("utf-8"))

    operators = {operator["id"]: operator for operator in operators}
    return operators


class Verifier:
    def __init__(self, app_name, base_url):
        self.threshold_percent = 40
        self.app_name = app_name
        self.base_url = base_url
        self.operators = get_operators()
        self.aggregated_public_key = attestation.new_zero_g2_point()
        for operator in self.operators.values():
            self.aggregated_public_key += operator["public_key_g2"]

    def verify_signature(self, message, signature_hex, nonsigners):
        total_stake = sum([operator["stake"] for operator in self.operators.values()])
        nonsigners = [self.operators[_id] for _id in nonsigners]
        nonsigners_stake = sum([operator["stake"] for operator in nonsigners])
        if 100 * nonsigners_stake / total_stake > 100 - self.threshold_percent:
            return False

        public_key = self.aggregated_public_key
        for operator in nonsigners:
            public_key -= operator["public_key_g2"]

        signature = attestation.new_zero_signature()
        signature.setStr(signature_hex.encode("utf-8"))

        message = hash(message)
        return signature.verify(public_key, message.encode("utf-8"))

    def get_last_finalized(self):
        resp = requests.get(
            f"{self.base_url}/node/{self.app_name}/batches/finalized/last"
        )
        data = resp.json()["data"]
        message = json.dumps(
            {
                "app_name": data["app_name"],
                "state": "locked",
                "index": data["index"],
                "hash": data["hash"],
                "chaining_hash": data["chaining_hash"],
            },
            sort_keys=True,
        )

        signature = data["finalization_signature"]
        nonsigners = data["nonsigners"]
        result = self.verify_signature(
            message,
            signature,
            nonsigners,
        )
        assert result, "last finalized verification failed"
        print(f"app: {data['app_name']}, index: {data['index']}, verification result: {result}")
        return data

    def get_finalized_after(self, index, chaining_hash):
        data = self.get_last_finalized()
        if data["index"] <= index:
            return chaining_hash, []

        checked_batches = []
        while True:
            resp = requests.get(
                f"{self.base_url}/node/{self.app_name}/batches/finalized?after={index}"
            )
            batches = resp.json()["data"]
            for batch in batches:
                index += 1
                chaining_hash = hash(chaining_hash + hash(batch))
                checked_batches.append(batch)
                if index == data["index"]:
                    assert (
                        chaining_hash == data["chaining_hash"]
                    ), "invalid chaining_hash"
                    return chaining_hash, checked_batches

    def batches(self):
        # todo: support custom index to start with
        base_data = self.get_last_finalized()
        index = base_data["index"]
        chaining_hash = base_data["chaining_hash"]

        while True:
            chaining_hash, batches = self.get_finalized_after(index, chaining_hash)
            for batch in batches:
                yield batch, index
                index += 1


if __name__ == "__main__":
    operators = get_operators()
    base_url = random.choice(list(operators.values()))["socket"]
    print(base_url)
    verifier = Verifier("simple_app", base_url)
    for batch, index in verifier.batches():
        txs = json.loads(batch)
        for i, tx in enumerate(txs):
            print(index, i, tx)
