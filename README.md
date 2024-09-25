# zellular.py

A Python SDK for interacting with the Zellular network.

## Dependencies

To use zelluar.py, you'll need to install the [MCL](https://github.com/herumi/mcl) native library. Follow these steps to install it:

```
$ sudo apt install libgmp3-dev
$ wget https://github.com/herumi/mcl/archive/refs/tags/v1.93.zip
$ unzip v1.93.zip
$ cd mcl-1.93
$ mkdir build
$ cd build
$ cmake ..
$ make
$ make install
```

## Installation

Install the zelluar.py package via pip:

```
pip install zellular
```

## Usage

### Getting Nodes

Zellular Testnet is deployed on the EigenLayer Holesky network. You can query the list of nodes using the following code:

```python
from pprint import pprint
import zellular

operators = zellular.get_operators()
pprint(operators)
```
Example output:

```
{'0x3eaa...078c': {
    'id': '0x3eaa...078c',
    'operatorId': '0xfd17...97fd',
    'pubkeyG1_X': '1313...2753',
    'pubkeyG1_Y': '1144...6864',
    'pubkeyG2_X': ['1051...8501', '1562...5720'],
    'pubkeyG2_Y': ['1601...1108', '1448...1899'],
    'public_key_g2': <eigensdk.crypto.bls.attestation.G2Point object at 0x7d8f31b167d0>,
    'socket': 'http://5.161.230.186:6001',
    'stake': 1
}, ... }
```

> [!TIP]
> The node URL of each operator can be accessed using `operator["socket"]`.

### Posting Transactions

Zellular sequences transactions in batches. You can send a batch of transactions like this:

```python
import requests
from uuid import uuid4
import time
from zellular import Zellular

base_url = "http://5.161.230.186:6001"
app_name = "simple_app"

zellular = Zellular(app_name, base_url)

t = int(time.time())
txs = [{"operation": "foo", "tx_id": str(uuid4()), "t": t} for _ in range(5)]

index = zellular.send(txs, blocking=True)
```

When setting `blocking=True`, the method waits for the batch to be sequenced and returns the index.

> [!TIP]
> You can add your app to zellular test network using:
> `curl -X POST https://www.zellular.xyz/testnet/apps -H "Content-Type: application/json" -d '{"app_name": "your-app-name"}'`


### Fetching and Verifying Transactions

Unlike reading from a traditional blockchain, where you must trust the node you're connected to, Zellular allows trustless reading of sequenced transactions. This is achieved through an aggregated BLS signature that verifies if the sequence of transaction batches is approved by the majority of Zellular nodes. The Zellular SDK abstracts the complexities of verifying these signatures, providing a simple way to constantly pull the latest finalized transaction batches:

```python
import json
from zellular import Zellular

base_url = "http://5.161.230.186:6001"
app_name = "simple_app"

zellular = Zellular(app_name, base_url)

for batch, index in zellular.batches(after=0):
    txs = json.loads(batch)
    for i, tx in enumerate(txs):
        print(index, i, tx)
```
Example output:

```
app: simple_app, index: 1, result: True
app: simple_app, index: 2, result: True
583 0 {'tx_id': '7eaa...2101', 'operation': 'foo', 't': 1725363009}
583 1 {'tx_id': '5839...6f5e', 'operation': 'foo', 't': 1725363009}
583 2 {'tx_id': '0a1a...05cb', 'operation': 'foo', 't': 1725363009}
583 3 {'tx_id': '6339...cc08', 'operation': 'foo', 't': 1725363009}
583 4 {'tx_id': 'cf4a...fc19', 'operation': 'foo', 't': 1725363009}
...
```

If you want to start reading batches from the latest finalized batch rather than from the beginning, you can achieve this by specifying the `after` parameter with the latest index. Here’s an example of how to do this:

```python
index = zellular.get_last_finalized()["index"]
for batch, index in zellular.batches(after=index):
    ...
```