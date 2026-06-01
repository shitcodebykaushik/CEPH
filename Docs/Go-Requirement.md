# The Go 
- The go is one of the most concurrent programming language while the ceph is the most scalable storage backends available .
- Multi-tenancy means multiple independent users are running their workloads on the exact same physical bare-metal hardware simultaneoulsly ,but they are completrly isolated from each other .

- `Orchestrator` The master controller software (the brain) . (It is the application written in go). it listens to the API requests,decides which physical server has enough room to host a new VM, provisions the storage,wires up the networking and spins the vm up or down .

- `Network Isolation` The multi-Tenant part , we create the custom  docker internal bridge network(e.g cloud-net). All containers get an internal IP address(127.18.0.2) . They can talk to each other perfectly over this virtual network . But they are completely hidden and secure from your public wifi network . 
- 

