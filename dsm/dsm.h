#ifndef DSM_H
#define DSM_H
#include <stdint.h>
#include <stdbool.h>
void change_access(uintptr_t addr, int NEW_PROT);
void *get_page(uintptr_t addr);
void set_page(uintptr_t addr, void *page_copy);
void setup(int num_pages, int index, int total_servers, bool call_tests);

void test_legal_read(int num_pages, int index, int total_servers, char*p);
void test_legal_write(int num_pages, int index, int total_servers, char *p);
void test_illegal_read(int num_pages, int index, int total_servers, char *p);
void test_illegal_write(int num_pages, int index, int total_servers, char *p);

// #ifdef __cplusplus
// extern "C" {
// #endif

// void *MakeClient(const char *peers, const char *central, int me);
// char *HandleRead(void *clientPtr, uintptr_t addr);
// void HandleWrite(void *clientPtr, uintptr_t addr);

// #ifdef __cplusplus
// }
// #endif
#endif