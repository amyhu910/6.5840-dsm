// #include <stdlib.h>
// #include <stdio.h>
// #include <errno.h>
// #include <unistd.h>
// #include <string.h>
// #include <signal.h>
// #include <stdint.h>
// #include <sys/mman.h>
// #include <sys/resource.h>
// #include <math.h>

// static size_t page_size;

// // align_down - rounds a value down to an alignment
// // @x: the value
// // @a: the alignment (must be power of 2)
// //
// // Returns an aligned value.
// #define align_down(x, a) ((x) & ~((typeof(x))(a) - 1))

// #define AS_LIMIT	(1ULL << 40) // Maximum limit on virtual memory bytes
// #define MAX_SQRTS	(1 << 27) // Maximum limit on sqrt table entries
// static double *sqrts;

// // Use this helper function as an oracle for square root values.
// static void
// calculate_sqrts(double *sqrt_pos, int start, int nr)
// {
//   int i;

//   for (i = 0; i < nr; i++)
//     sqrt_pos[i] = sqrt((double)(start + i));
// }

// static void
// handle_sigsegv(int sig, siginfo_t *si, void *ctx)
// {
//   // Your code here.
//   static double *va = NULL;
//   int i, pos;
//   uintptr_t pg;

//   pg = align_down((uint64_t)si->si_addr, page_size);
//   if (va)
//     munmap(va, page_size);
//   va = mmap((void *)pg, page_size, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);

//   if (pg <= (uint64_t)sqrts) {
//     pos = 0;
//   } else {
//     pos = (pg - (uint64_t)sqrts) / 8;
//   }

//   for (i = 0; i < page_size / 8; i++) {
//     calculate_sqrts(&va[i], pos + i, 1);
//   }
// }

// static void
// setup_sqrt_region(void)
// {
//   struct rlimit lim = {AS_LIMIT, AS_LIMIT};
//   struct sigaction act;

//   // Only mapping to find a safe location for the table.
//   sqrts = mmap(NULL, MAX_SQRTS * sizeof(double) + AS_LIMIT, PROT_NONE,
// 	       MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
//   if (sqrts == MAP_FAILED) {
//     fprintf(stderr, "Couldn't mmap() region for sqrt table; %s\n",
// 	    strerror(errno));
//     exit(EXIT_FAILURE);
//   }

//   // Now release the virtual memory to remain under the rlimit.
//   if (munmap(sqrts, MAX_SQRTS * sizeof(double) + AS_LIMIT) == -1) {
//     fprintf(stderr, "Couldn't munmap() region for sqrt table; %s\n",
//             strerror(errno));
//     exit(EXIT_FAILURE);
//   }

//   // getrlimit(RLIMIT_AS, &lim);

//   // Set a soft rlimit on virtual address-space bytes.
//   if (setrlimit(RLIMIT_AS, &lim) == -1) { // only works for 1 << 40 and up
//     fprintf(stderr, "Couldn't set rlimit on RLIMIT_AS; %s\n", strerror(errno));
//     exit(EXIT_FAILURE);
//   }

//   // Register a signal handler to capture SIGSEGV.
//   act.sa_sigaction = handle_sigsegv;
//   act.sa_flags = SA_SIGINFO;
//   sigemptyset(&act.sa_mask);
//   if (sigaction(SIGSEGV, &act, NULL) == -1) {
//     fprintf(stderr, "Couldn't set up SIGSEGV handler;, %s\n", strerror(errno));
//     exit(EXIT_FAILURE);
//   }
// }

// static void
// test_sqrt_region(void)
// {
//   int i, pos = rand() % (MAX_SQRTS - 1);
//   double correct_sqrt;

//   printf("Validating square root table contents...\n");
//   srand(0xDEADBEEF);

//   for (i = 0; i < 500000; i++) {
//     if (i % 2 == 0)
//       pos = rand() % (MAX_SQRTS - 1);
//     else
//       pos += 1;
//     calculate_sqrts(&correct_sqrt, pos, 1);
//     if (sqrts[pos] != correct_sqrt) {
//       fprintf(stderr, "Square root is incorrect. Expected %f, got %f.\n",
//               correct_sqrt, sqrts[pos]);
//       exit(EXIT_FAILURE);
//     }
//   }

//   printf("All tests passed!\n");
// }

// // int
// // main(int argc, char *argv[])
// // {
// //     struct rlimit limits;

// //     // Get the current resource limits for RLIMIT_AS (address space)
// //     if (getrlimit(RLIMIT_AS, &limits) == -1) {
// //         // Error handling if getrlimit fails
// //         perror("getrlimit");
// //         return 1;
// //     }

// //     // Output the current soft limit for address space
// //     printf("Current soft limit for address space: %llu\n", limits.rlim_cur);
// //     printf("Current hard limit for address space: %llu\n", limits.rlim_max);  
// //   page_size = sysconf(_SC_PAGESIZE);
// //   printf("page_size is %ld\n", page_size);
// //   setup_sqrt_region();
// //   test_sqrt_region();
// //   return 0;
// // }