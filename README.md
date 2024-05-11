# 6.5840-dsm
A DSM is a distributed shared memory. It refers to a method used in parallel computing where multiple machines share a common memory space, allowing them to access and manipulate shared data as if it were stored locally. 

This DSM implementation is based on the IVY design, but the distributed manager version. There is no longer a central server bottleneck and instead the clients dynamically are assigned ownership and manage system metadata.

Each page in the shared memory is “owned” by a specific client server. At any given point in time, a page can either be 1. read-write on exactly one machine and invalid on all others or 2. read-only on any number of machines. The library will handle all page faults and grant the necessary permissions to complete an operation. 

To run your own functions, import your C code into the `dsm` folder (see `matmul.c` for an example). Call the relevant C functions in the `ClientSetup` function in `dsm/client.go`.

To start running the DSM, import all code to the relevant machines. Compile at each machine using `go build`. Then, if you are to run the code with two clients with IP addresses `ip1`, and `ip2` respectively, you can run the following commands at each machine to create a DSM with `numpages` pages.

For client 1:
```bash
./6.5840-dsm -p 0 2 numpages ip1
```

For client 2: 
```bash
./6.5840-dsm -p 1 2 numpages ip2
```

To get help, try the following command:
```bash
./6.5840-dsm -h
```