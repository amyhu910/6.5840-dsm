#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <signal.h>
#include <stdint.h>
#include <sys/mman.h>
#include <sys/resource.h>
#include <math.h>

#include "dsm.h"
#include "_cgo_export.h"

#define page_size sysconf(_SC_PAGESIZE)

// align_down - rounds a value down to an alignment
// @x: the value
// @a: the alignment (must be power of 2)
//
// Returns an aligned value.

void *align_down(void *addr) {
    return (void *)((uintptr_t)addr & ~(page_size - 1));
}

static void
handle_sigsegv(int sig, siginfo_t *info, void *ctx)
{
    uintptr_t pg;

    pg = (uintptr_t)align_down((void *) info->si_addr);
    unsigned long pte;

    if (mincore((void *)info->si_addr, page_size, (char *)&pte) == -1) {
        perror("mincore");
        return;
    }

    int prot = pte & (PROT_READ | PROT_WRITE);

    if (prot & PROT_READ) {
        memcpy(&pg, HandleRead((uintptr_t) info->si_addr), page_size);
    } else {
        HandleWrite((uintptr_t) info->si_addr);
    }
}

void
setup(int num_pages, int index, int total_servers) {
    // set up sigsegv handler
    // MakeClient("localhost:8080", "localhost:8081", index);
    struct sigaction act;
    act.sa_sigaction = handle_sigsegv;
    act.sa_flags = SA_SIGINFO;
    sigemptyset(&act.sa_mask);
    if (sigaction(SIGSEGV, &act, NULL) == -1) {
        fprintf(stderr, "Couldn't set up SIGSEGV handler;, %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    // map all pages as PROT_NONE
    char *p = mmap(NULL, num_pages * page_size, PROT_NONE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (p == MAP_FAILED) {
        fprintf(stderr, "Couldn't mmap memory; %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    int curpage = (int)floor((index / total_servers) * num_pages);
    int nextpage = (int)floor(((index + 1) / total_servers) * num_pages);

    for (int i = curpage; i < nextpage; i++) {
        // set up the page at index to be read/write
        mprotect(p + i * page_size, page_size, PROT_READ | PROT_WRITE);
    }
}

void 
change_access(uintptr_t addr, int NEW_PROT) {
    // set up the page at index to be read-only
    mprotect((void *)addr, page_size, NEW_PROT);
}

void *get_page(uintptr_t addr) {
    void *page_start = align_down((void *)addr);

    // Allocate memory to hold the page copy
    void *page_copy = malloc(page_size);
    if (page_copy == NULL) {
        fprintf(stderr, "Failed to allocate memory for page copy.\n");
        return NULL;
    }

    // Copy the contents of the page into the allocated memory
    memcpy(page_copy, page_start, page_size);

    return page_copy;
}

void set_page(uintptr_t addr, void *data) {
    void *page_start = align_down((void *)addr);
    uintptr_t offset = (uintptr_t)addr - (uintptr_t)page_start;
    memcpy(page_start + offset, data, page_size);
}

// int main(int argc, char **argv) {
//     if (argc != 4) {
//         fprintf(stderr, "Usage: %s <num_pages> <index> <total_servers>\n", argv[0]);
//         return 1;
//     }

//     int num_pages = atoi(argv[1]);
//     int index = atoi(argv[2]);
//     int total_servers = atoi(argv[3]);
//     // int num_pages = 1;
//     // int index = 0;
//     // int total_servers = 2;

//     setup(num_pages, index, total_servers);

//     while (1) {
//         sleep(1);
//     }

//     return 0;
// }