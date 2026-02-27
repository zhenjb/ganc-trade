# Ganc
*(update)*


## ©️ Usage
### Clone Repository & Directory 
```bash
git clone https://github.com/zjqingzun/scyl-Ganc.git
cd scyl-Ganc
```

### Check Environment & Package (Go, Ignite, Rustup, Circom 2)
```bash
# go
go version

# ignite
ignite version

# cargo (Rust)
cargo --version

# circom
circom --version
```
If you haven't installed Ignite CLI yet, please refer to the official [Ignite CLI installation guide](./cli/readme.md). <br>

If you haven't installed Circom 2, execute the following commands:
```bash
# Rustup (optional)
curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh
source "$HOME/.cargo/env"
cargo --version

# Circom 2
git clone https://github.com/iden3/circom.git
cd circom
cargo build --release
cargo install --path circom
circom --version

circom --help
```

### Run Chain
```bash
cd sw/ob
ignite chain serve -r
```

#### Run Node (Ob Node)
***Mandatory:*** 
*The prerequisite is that the chain must be running first.*
```bash
# Open a new terminal
cd sw/ob
obd tx dex --help
```

#### Test Matching Orderbook
```bash
# Open a new terminal 
cd sw/sh-scyl/test/
bash order-matching@10S10B-empty-sort.sh
```


## ©️ Terms and Privacy Policy
We hereby notify you that your use of or involvement with this project constitutes your acceptance of the following Terms and Privacy Policy: <br>

- ___Acknowledgment of Terms and Usage Policies for Ignite CLI:__ We acknowledge and adhere to the terms of service and usage policies of the third-party software provider, [Ignite CLI](https://ignite.com/)._ <br>
- ___Acknowledgment of Terms for Circom 2 and SNARK JS:__ We recognize and comply with the terms and conditions governing the source code management and usage of [Circom 2](https://iden3.io/circom) and [snarkjs](https://github.com/iden3/snarkjs)._ <br>
- ___Contributions of Applied Products:__ We acknowledge the significant contributions of the various tools and libraries utilized in this project._ <br>


## ©️ Contribution 
*(update)*


## ©️ Security
*(update)*


## ©️ License
Copyright © 2026 zjDSPF <br>
<img src="public/icon/zjDSPF.png" alt="zjDSPF" width="150" height="150"> <br>

