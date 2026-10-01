# gotron internals

Documentation for people working on gotron itself. To use the library, start with the
[README](../README.md) and the [integration guide](../skills/gotron/SKILL.md).

| Document                           | What it covers                                                                                                                                                        |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [architecture.md](architecture.md) | Layers, how `client.New` wires the transport stack, error internals, design rules                                                                                     |
| [transport.md](transport.md)       | The `Transport` interface and its five implementations, decoding java-tron's HTTP JSON, accepted gRPC/HTTP differences, the add-an-RPC-method checklist, endpoint map |
| [testing.md](testing.md)           | Unit tests, public-node integration tests, `synctest` health tests, the local private network and the gRPC/HTTP parity suite                                          |
