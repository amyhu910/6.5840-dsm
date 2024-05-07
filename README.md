# 6.5840-dsm
A DSM is a distributed shared memory. It refers to a method used in parallel computing where multiple machines share a common memory space, allowing them to access and manipulate shared data as if it were stored locally. 

This DSM implementation is based on the IVY design, in which there is one central server managing metadata and a number of client servers that can independently access the shared memory. 

In particular, pages are pre-allocated to machines during the setup phase. Each machine is assigned an index which will determine which set of pages it is assigned during initialization. At any given point in time, a page can either be 1. read-write on exactly one machine and invalid on all others or 2. read-only on any number of machines. The library will handle all page faults and grant the necessary permissions to complete an operation. 

To run your own functions, import your C code into the `dsm` folder. Call the relevant C functions in the `ClientSetup` function in `dsm/client.go`.

To start running the DSM, import all code to the relevant machines. Compile at each machine using `go build`. Then, if you are to run the code with one central server and two clients with IP addresses `ip0`, `ip1`, and `ip2` respectively, you can run the following commands at each machine to create a DSM with `numpages` pages.

For the central server:
```bash
./6.5840-dsm -c numpages ip1 ip2
```

For client 1:
```bash
./6.5840-dsm -p 0 2 numpages ip0
```

For client 2: 
```bash
./6.5840-dsm -p 1 2 numpages ip0
```

To get help, try the following command:
```bash
./6.5840-dsm -h
```