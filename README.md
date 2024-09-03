# zellular.py

a python sdk for zellular

## Dependencies

It required to [MCL](https://github.com/herumi/mcl) native package to be installed.
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

```
pip install zellular
```

## Example

### Getting Nodes

```python
>>> from pprint import pprint
>>> import zellular
>>> operators = zellular.get_operators()
>>> pprint(operators)
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

### Posting

```python
>>> import requests
>>> import time
>>> base_url = "http://5.161.230.186:6001"
>>> app_name = "simple_app"
>>> t = int(time.time())
>>> txs = [{ "operation": "foo", "tx_id": str(uuid4()), "t": t } for i in range(5)]
>>> resp = requests.put(f"{base_url}/node/{app_name}/batches", json=txs)
>>> print(resp.status_code)
```

### Fetching and Verifying

```python
{"data":{},"message":"The batch is received successfully.","status":"success"}
>>> import json
>>> import zellular
>>> verifier = zellular.Verifier("simple_app", "http://5.161.230.186:6001")
>>> for batch, index in verifier.batches():
...     txs = json.loads(batch)
...     for i, tx in enumerate(txs):
...         print(index, i, tx)

app: simple_app, index: 481238, result: True
app: simple_app, index: 481240, result: True
583 0 {'tx_id': '7eaa...2101', 'operation': 'foo', 't': 1725363009}
583 1 {'tx_id': '5839...6f5e', 'operation': 'foo', 't': 1725363009}
583 2 {'tx_id': '0a1a...05cb', 'operation': 'foo', 't': 1725363009}
583 3 {'tx_id': '6339...cc08', 'operation': 'foo', 't': 1725363009}
583 4 {'tx_id': 'cf4a...fc19', 'operation': 'foo', 't': 1725363009}
...
```
