# Ganc
*(update)*


## ©️ Usage
### Clone Repository & Directory 
```bash
git clone https://github.com/zjqingzun/scyl-Ganc.git
cd scyl-Ganc
```

<details>
  <summary>Check Environment & Package (Go, Ignite, Rustup, Circom 2)</summary>

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
</details>

### Ganc Setup 
```bash
chmod +x exe.sh
./exe.sh
```

### Run Chain
```bash
ganc chain
```

<details>
  <summary>Run Node (Ob Node) - Ganc v0.0.x</summary>

***Mandatory:*** 
*The prerequisite is that the chain must be running first.*
```bash
# Open a new terminal
ganc test -obs node@tx
```
</details>

### Ganc v0.1.0
#### -ob
<details>
  <summary>Test Matching Orderbook</summary>

***Mandatory:*** 
*The prerequisite is that the chain must be running first.* <br>
```bash
# Open a new terminal 
ganc test -ob matching@06S06B
```
or 
```bash
# Open a new terminal 
ganc test -ob matching@10S10B
```
The order was perfectly matched.
</details>

<details>
    <summary>Check Escrow and Selfement</summary>

***Mandatory:*** 
*The prerequisite is that the chain must be running first.* 
```bash
# Open a new terminal
ganc test -ob matching@balance
```
</details>

#### -obs
<details>
    <summary>Deposit Token</summary>

***Mandatory:*** 
*The prerequisite is that the chain must be running first.* 
```bash
# Open a new terminal
ganc test -obs smartc@deposit
```
</details>

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

