## GalaxyNetWork(GNC) based Go Ethereum v1.10.8

## Changes made:
   * Consensus algorithm: (As reference) Chia Proof of Space Construction. 
      https://www.chia.net/assets/Chia_Proof_of_Space_Construction_v1.1.pdf
   * Geth v1.10.8 is a pre-announced hotfix release to patch a vulnerability in the EVM (CVE-2021-39137).
   * Build implementation based Geth v1.10.8 version
   * The address balance will be mapped to the mainnet
   * Smart Contracts supported
   * Economic model upgraded

### public rpc/api
http://chain-node.galaxynetwork.vip

### Notice
* 1.The address showed to users shall keep the prefix with 'GNC';
* 2.Before calling the GalaxyNetwork RPC interface, please replace the transfer address that users entering from prefix 'GNC' to '0x'.
* 3.Keep the RPC/API same with Ethereum v1.10.8

### Shown to users:
```js
    var Web3 = require('web3');

    var web3 = new Web3(new Web3.providers.HttpProvider("http://chain-node.galaxynetwork.vip"));

    var newAccount=web3.eth.accounts.create()

    // 0x6bacec0a630a53fdbae5f1f10bf87fe2b422eec1  -> GNC6bacec0a630a53fdbae5f1f10bf87fe2b422eec1
    console.log(newAccount.address.replace('0x','GNC'))
    console.log(newAccount.privateKey)
```

### User input:
```js
    //user input GNC address
    var oldGNC_Address ='GNC6cBe9DF6DF54281D363e7a5e1790dc66212438C7'

    //GNC7d4ed9084A364424d1087d26C4Ef092EAfB0b395 -> 0x7d4ed9084A364424d1087d26C4Ef092EAfB0b395
    var newGNC_Address=oldGNC_Address.replace('GNC','0x')
    
    //call rpc ...
    web3.eth.getBalance(newGNCAddress).then(
        console.log
    )
```

### Transaction Demo
* https://github.com/gnc-project/galaxynetwork-web3js
* https://github.com/gnc-project/galaxynetwork-web3j

## Mining
```shell
                                       _____  miner 1 (reserved space)
                                      /
 GalaxyNetWork peers  --------   Geth Node peer  ------  miner 2 (reserved space)
                                      \_____  miner 3 (reserved space)
```

\
\
\
\
\
&NewLine;


Automated builds are available for stable releases and the unstable master branch. Binary
archives are published at https://github.com/gnc-project/galaxynetwork.

## Building the source

Building `geth` requires both a Go (version 1.14 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make geth
./geth --http.api='eth,web3,net,debug' --http.port=8545 --gcmode archive
```

or, to build the full suite of utilities:

```shell
make all
```

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.
