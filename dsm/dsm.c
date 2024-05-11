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

// align_down - rounds a value down to an alignment
// @x: the value
// @a: the alignment (must be power of 2)
//
// Returns an aligned value.
char* p;

void *align_down(void *addr) {
    return (void *)((uintptr_t)addr & ~(PAGE_SIZE - 1));
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

    if (mincore((void *)pg, PAGE_SIZE, (char *)&pte) == -1) {
        perror("mincore");
        return;
    }

    int prot = pte & (PROT_READ | PROT_WRITE);

    if (prot & PROT_READ) {
        HandleWrite((uintptr_t) pg -  (uintptr_t) p);
    } else {
        HandleRead((uintptr_t) pg - (uintptr_t) p);
    }
}

void setup_handler() {
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
}
void make_all_pages_accesible(int num_pages) {
    mprotect(p, num_pages * PAGE_SIZE, PROT_READ | PROT_WRITE);
}

void create_pages(int num_pages) {
    p = mmap(NULL, num_pages * PAGE_SIZE, PROT_NONE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (p == MAP_FAILED) {
        fprintf(stderr, "Couldn't mmap memory; %s\n", strerror(errno));
        exit(EXIT_FAILURE);
    }
    make_all_pages_accesible(num_pages)
    for (int i = 0; i < num_pages; i++) {
        *(p+i*PAGE_SIZE) = i;
    }
}

void
setup(int num_pages, int index, int total_servers) {
    // set up sigsegv handler
    setup_handler();

    // map all pages as PROT_NONE
    printf("Mapping all pages as PROT_NONE\n");
    create_pages(num_pages);
    
    // Give client 0 initial ownership of all pages
    if (index == 0) {
        make_all_pages_accesible(num_pages);
    }
}

void test_one_client(int num_pages, int index, int total_servers) {
    printf("Testing one client\n");
    if (index == 1) {
        test_legal_read(num_pages, index, total_servers);
        test_legal_write(num_pages, index, total_servers);
        // test_illegal_read(num_pages, index, total_servers);
        // test_illegal_write(num_pages, index, total_servers);
        test_illegal_read_misaligned(num_pages, index, total_servers);
        test_illegal_write_misaligned(num_pages, index, total_servers);
        test_invalid_illegal_write(num_pages, index, total_servers);
        printf("All tests passed\n");
    }
}

void test_concurrent_clients(int num_pages, int index, int total_servers) {
    printf("Testing concurrent clients\n");
    test_illegal_read_concur(num_pages, index, total_servers);
    test_illegal_write_concur(num_pages, index, total_servers);
}

void test_legal_read(int num_pages, int index, int total_servers) {
    printf("Testing legal read\n");
    int curpage = (int)floor((index / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE);

    // Dereference the pointer to read the value
    printf("Deferencing pointer\n");
    int value = *ptr;
    printf("Value: %d\n", value);
    printf("Legal read passed with no errors\n");
    // No errors should occur
}

void test_legal_write(int num_pages, int index, int total_servers) {
    printf("Testing legal write\n");
    int curpage = (int)floor((index / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE);

    // Dereference the pointer to write the value
    printf("Deferencing pointer\n");
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Legal write passed with no errors\n");
    // No errors should occur
}

void test_illegal_read(int num_pages, int index, int total_servers) {
    printf("Testing illegal read\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE);

    // Dereference the pointer to read the value
    printf("Deferencing pointer\n");
    int value = *ptr;
    printf("Value: %d\n", value);
    printf("Illegal read passed with no errors\n");
    // Segfault should be called
}

void test_illegal_read_misaligned(int num_pages, int index, int total_servers) {
    printf("Testing illegal read\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE + 1);

    // Dereference the pointer to read the value
    printf("Deferencing pointer\n");
    int value = *ptr;
    printf("Value: %d\n", value);
    printf("Illegal read passed with no errors\n");
    // Segfault should be called
}

void test_illegal_write(int num_pages, int index, int total_servers) {
    printf("Testing illegal write\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE);

    // Dereference the pointer to write the value
    printf("Deferencing pointer\n");
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Illegal write passed with no errors\n");
    // No errors should occur
}

void test_illegal_write_misaligned(int num_pages, int index, int total_servers) {
    printf("Testing illegal write\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages);

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE + 1);

    // Dereference the pointer to write the value
    printf("Deferencing pointer\n");
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Illegal write passed with no errors\n");
    // No errors should occur
}

void test_invalid_illegal_write(int num_pages, int index, int total_servers) {
    // TODO: Currently assumes that each clients has at least 2 pages allocated to it
    printf("Testing illegal write to invalid page\n");
    int wrongindex = (index + 1) % total_servers;
    int curpage = (int)floor((wrongindex / (double)total_servers) * num_pages) + 1;

    printf("Getting pointer\n");
    int* ptr = (int*)(p + curpage * PAGE_SIZE);

    // Dereference the pointer to write the value
    printf("Deferencing pointer\n");
    *ptr = 42;
    printf("Value: %d\n", *ptr);
    printf("Illegal write passed with no errors\n");
    // No errors should occur
}

void test_illegal_read_concur(int num_pages, int index, int total_servers) {
    printf("Testing illegal read concurrently\n");
    for (int i = 0; i < num_pages; i++) {
        // set up the page at index to be read/write
        int* ptr = (int*)(p + i * PAGE_SIZE);

        // Dereference the pointer to read the value
        int value = *ptr;
        printf("Value: %d\n", value);
    }
    printf("All concurrent read tests passed\n");
}

void test_illegal_write_concur(int num_pages, int index, int total_servers) {
    printf("Testing illegal write concurrently\n");
    for (int i = 0; i < num_pages; i++) {
        // set up the page at index to be read/write
        int* ptr = (int*)(p + i * PAGE_SIZE);

        // Dereference the pointer to read the value
        *ptr = 10 * index + 1;
        printf("Value: %d\n", *ptr);
    }
    printf("All concurrent write tests passed\n");
}

void change_access(uintptr_t addr, int NEW_PROT) {
    printf("Changing access to %i\n", NEW_PROT);
    // set up the page at index to be read-only
    mprotect((void *)p + addr, PAGE_SIZE, NEW_PROT);
}

void *get_page(uintptr_t addr) {
    void *page_start = get_pa((void *)addr);
    printf("Getting page %p: %d\n", page_start, *(int *)page_start);

    // Allocate memory to hold the page copy
    void *page_copy = malloc(PAGE_SIZE);
    if (page_copy == NULL) {
        fprintf(stderr, "Failed to allocate memory for page copy.\n");
        return NULL;
    }

    // Copy the contents of the page into the allocated memory
    memcpy(page_copy, page_start, PAGE_SIZE);

    return page_copy;
}

void set_page(uintptr_t addr, void *data) {
    void *page_start = get_pa((void *)addr);
    printf("Setting page %p\n", page_start);
    mprotect((void *)page_start, PAGE_SIZE, PROT_WRITE);
    memcpy(page_start, data, PAGE_SIZE);
}