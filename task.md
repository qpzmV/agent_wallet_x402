我的AI web3 agent wallet，请你给我设计一个代付gas的功能，用x402实现, 我想象的架构图如下：
Agent
 │
 │ POST /execute
 ▼
x402 middleware
 │
 │ payment verified
 ▼
Execution Engine
 │
 ├───────────────┬───────────────┬───────────────┐
 ▼               ▼               ▼
EVM             Solana          Sui
 │               │               │
 │ sponsor ETH   │ sponsor SOL   │ sponsor gas coin
 ▼               ▼               ▼
Blockchain     Blockchain      Blockchain
要求：
1.x402是一个单独的server
2.执行引擎是另一个后端server
3.x402和执行引擎都能否用go实现
4.有测试代码，对接eth,solana,sui的3条测试链
5.代付功能可否也是一个单独的微服务，不管user是构造什么的交易，我们都能帮其付gas
6.代付gas，收用户的usdc，拿一部分作为手续费，这也是我们产品盈利点