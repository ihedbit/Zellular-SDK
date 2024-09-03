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

### Getting list of nodes

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


### Fetching & Verifying

```python
>>> import json
>>> import zellular
>>> verifier = zellular.Verifier("simple_app", "http://5.161.230.186:6001")
>>> for batch, index in verifier.batches():
...     txs = json.loads(batch)
...     for i, tx in enumerate(txs):
...         print(index, i, tx)

app: simple_app, index: 481237, result: True
app: simple_app, index: 481238, result: True
481237 0 {'tx_id': '391e...f4c9', 'operation': 'foo', 't': 1725351862}
app: simple_app, index: 481238, result: True
app: simple_app, index: 481240, result: True
481238 0 {'tx_id': '96df...0452', 'operation': 'foo', 't': 1725351863}
481239 0 {'tx_id': '95ac...ef2e', 'operation': 'foo', 't': 1725351863}
...
```
