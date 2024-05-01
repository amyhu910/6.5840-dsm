#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <signal.h>
#include <stdint.h>
#include <stdbool.h>
#include <sys/mman.h>
#include <sys/resource.h>
#include <math.h>

#include "dsm.h"
#include "_cgo_export.h"

#define page_size sysconf(_SC_PAGESIZE)

char *p;

// align_down - rounds a value down to an alignment
// @x: the value
// @a: the alignment (must be power of 2)
//
// Returns an aligned value.

void *align_down(void *addr) {
    return (void *)((uintptr_t)addr & ~(page_size - 1));
}

void *get_pa(void *va) {
    return (void *)(((uintptr_t)p + (uintptr_t)va));
}

static void
handle_sigsegv(int sig, siginfo_t *info, void *ctx)
{
    printf("Handling SIGSEGV\n");
    uintptr_t pg;

    pg = (uintptr_t)align_down((void *) info->si_addr);
    unsigned long pte;

    if (mincore((void *)info->si_addr, page_size, (char *)&pte) == -1) {
        perror("mincore");
        return;
    }

    int prot = pte & (PROT_READ | PROT_WRITE);

    if (prot & PROT_READ) {
        HandleWrite((uintptr_t) info->si_addr -  (uintptr_t) p);
    } else {
        HandleRead((uintptr_t) info->si_addr - (uintptr_t) p);
    }
}

void
setup(int num_pages, int index, int total_servers, bool call_tests) {
    // set up sigsegv handler
    // MakeClient("localhost:8080", "localhost:8081", index);
    printf("Setting up SIGSEGV handler\n");
    struct sigaction actsegv;
    actsegv.sa_sigaction = handle_sigsegv;
    actsegv.sa_flags = SA_SIGINFO;
    sigemptyset(&actsegv.sa_mask);
    if (sigaction(SIGSEGV, &actsegv, NULL) == -1) {
        fprintf(stderr, "Couldn't set up SIGSEGV handler;, %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    printf("Setting up SIGBUS handler\n");
    struct sigaction actbus;
    actbus.sa_sigaction = handle_sigsegv;
    actbus.sa_flags = SA_SIGINFO;
    sigemptyset(&actbus.sa_mask);
    if (sigaction(SIGBUS, &actbus, NULL) == -1) {
        fprintf(stderr, "Couldn't set up SIGBUS handler;, %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    // map all pages as PROT_NONE
    printf("Mapping all pages as PROT_NONE\n");
    p = mmap(NULL, num_pages * page_size, PROT_NONE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (p == MAP_FAILED) {
        fprintf(stderr, "Couldn't mmap memory; %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }

    int curpage = (int)floor((index / (double)total_servers) * num_pages);
    int nextpage = (int)floor(((index + 1) / (double)total_servers) * num_pages);

    printf("Setting up pages %i through %i with read write permissions\n", curpage, nextpage);
    for (int i = curpage; i < nextpage; i++) {
        // set up the page at index to be read/write
        mprotect(p + i * page_size, page_size, PROT_READ | PROT_WRITE);
        int* ptr = (int*)(p + i * page_size);

        // Dereference the pointer to write the value
        *ptr = index;
        printf("Setting page %p: %d\n", p + i * page_size, *ptr);
    }

    if (call_tests) {
        test_legal_read(num_pages, index, total_servers);
        test_legal_write(num_pages, index, total_servers);
        test_illegal_read(num_pages, index, total_servers);
        test_illegal_write(num_pages, index, total_servers);
    }
}

void test_legal_read(int num_pages, int index, int total_servers) {
    printf("Testing legal read\n");
    int curpage = (int)floor((index / (double)total_servers) * num_pages);

    int* ptr = (int*)(p + curpage * page_size);

    // Dereference the pointer to read the value
    int value = *ptr;
    printf("Value: %d\n", value);
    printf("Legal read passed with no errors\n");
    // No errors should occur
}

void test_legal_write(int num_pages, int index, int total_servers) {
    printf("Testing legal write\n");
    int curpage = (int)floor((index / (double)total_servers) * num_pages);

    int* ptr = (int*)(p + curpage * page_size);

    // Dereference the pointer to write the value
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Legal write passed with no errors\n");
    // No errors should occur
}

void test_illegal_read(int num_pages, int index, int total_servers) {
    printf("Testing illegal read\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    int* ptr = (int*)(p + curpage * page_size);

    // Dereference the pointer to read the value
    int value = *ptr;
    printf("Value: %d\n", value);
    printf("Illegal read passed with no errors\n");
    // Segfault should be called
}

void test_illegal_write(int num_pages, int index, int total_servers) {
    printf("Testing illegal write\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    int* ptr = (int*)(p + curpage * page_size);

    // Dereference the pointer to write the value
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Illegal write passed with no errors\n");
    // No errors should occur
}
void 
change_access(uintptr_t addr, int NEW_PROT) {
    printf("Changing access to %i\n", NEW_PROT);
    // set up the page at index to be read-only
    mprotect((void *)p + addr, page_size, NEW_PROT);
}

void *get_page(uintptr_t addr) {
    void *page_start = get_pa((void *)addr);
    printf("Getting page %p: %d\n", page_start, *(int *)page_start);

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
    void *page_start = get_pa((void *)addr);
    printf("Setting page %p\n", page_start);
    mprotect((void *)page_start, page_size, PROT_WRITE);
    memcpy(page_start, data, page_size);
}