#ifndef DSM_H
#define DSM_H
#include <stdint.h>
void change_access(uintptr_t addr, int NEW_PROT);
void *get_page(uintptr_t addr);
void set_page(uintptr_t addr, void *page_copy);
#endif