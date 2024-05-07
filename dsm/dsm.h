#ifndef DSM_H
#define DSM_H
#include <stdint.h>
#include <stdbool.h>
#include <signal.h>
#define PAGE_SIZE sysconf(_SC_PAGESIZE)

extern char *p;
void create_pages(int num_pages);
void change_access(uintptr_t addr, int NEW_PROT);
void *get_page(uintptr_t addr);
void set_page(uintptr_t addr, void *page_copy);
void setup(int num_pages, int index, int total_servers);
void test_one_client(int num_pages, int index, int total_servers);
void test_concurrent_clients(int num_pages, int index, int total_servers);

void test_legal_read(int num_pages, int index, int total_servers);
void test_legal_write(int num_pages, int index, int total_servers);
void test_illegal_read(int num_pages, int index, int total_servers);
void test_illegal_write(int num_pages, int index, int total_servers);
void test_illegal_read_misaligned(int num_pages, int index, int total_servers);
void test_illegal_write_misaligned(int num_pages, int index, int total_servers);

void test_invalid_illegal_write(int num_pages, int index, int total_servers);
void test_illegal_read_concur(int num_pages, int index, int total_servers);
void test_illegal_write_concur(int num_pages, int index, int total_servers);

void *align_down(void *addr);
void *get_pa(void *va);
void setup_handler();

void setup_matmul(int num_pages, int index, int total_servers);
void multiply_matrices(int index, int total_servers);
void print_matrix(int row, int col, int* matrix);
#endif