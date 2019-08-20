#ifndef FYR_H
#define FYR_H

#define EXIT_FAILURE 1

#include <stddef.h>
#include <stdbool.h>
#include <stdint.h>

typedef uint8_t* addr_t;
typedef int32_t int_t;
typedef uint32_t uint_t;

typedef void (*fyr_dtr_t)(addr_t ptr);
typedef void (*fyr_dtr_arr_t)(addr_t ptr, int_t count);

addr_t fyr_alloc(int_t size);
addr_t fyr_alloc_arr(int_t count, int_t size);
void fyr_free(addr_t, fyr_dtr_t dtr);
void fyr_free_arr(addr_t, fyr_dtr_arr_t dtr);
bool fyr_isnull(addr_t);
void fyr_notnull_ref(addr_t);
addr_t fyr_incref(addr_t ptr);
#define fyr_incref_arr fyr_incref
void fyr_decref(addr_t ptr, fyr_dtr_t dtr);
void fyr_decref_arr(addr_t ptr, fyr_dtr_arr_t dtr);
addr_t fyr_lock(addr_t ptr);
void fyr_unlock(addr_t ptr, fyr_dtr_t dtr);
#define fyr_lock_arr fyr_lock
void fyr_unlock_arr(addr_t ptr, fyr_dtr_arr_t dtr);
int_t fyr_len_arr(addr_t ptr);
int_t fyr_len_str(addr_t ptr);
int_t fyr_min(int_t a, int_t b);
int_t fyr_max(int_t a, int_t b);
addr_t fyr_arr_to_str(addr_t array_ptr, addr_t data_ptr, int_t len);
void fyr_move_arr(addr_t dest, addr_t source, int_t count, int_t size, fyr_dtr_arr_t dtr);
bool fyr_cmp_ref(addr_t ptr1, addr_t ptr2);

#endif