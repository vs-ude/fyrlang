#include "fyr.h"
#include <stdlib.h>
#include <limits.h>
#include <assert.h>
#include <string.h>

// #include <stdio.h>
#define VALGRIND 1

addr_t fyr_alloc(int_t size) {
    // TODO: If int_t is larger than size_t, the size could be shortened.
    int_t* ptr = calloc(1, (size_t)size + 2 * sizeof(int_t));
    // printf("calloc %lx\n", (long)ptr);
    // No locks
    *ptr++ = 0;
    // One owner
    *ptr++ = 1;
    return (addr_t)ptr;
}

addr_t fyr_alloc_arr(int_t count, int_t size) {
    int_t* ptr = calloc(1, (size_t)count * (size_t)size + 3 * sizeof(int_t));
    // printf("calloc arr %lx\n", (long)ptr);
    // Number of elements in the array
    *ptr++ = count;
    // No locks
    *ptr++ = 0;
    // One owner
    *ptr++ = 1;
    return (addr_t)ptr;
}

void fyr_free(addr_t ptr, fyr_dtr_t dtr) {
    if (ptr == NULL) {
        return;
    }
    // Get locks
    int_t* lptr = ((int_t*)ptr) - 2;
    // Get reference count
    int_t* iptr = ((int_t*)ptr) - 1;
    // Only one reference remaining? -> object can be destroyed (unless it is locked)
    if (*iptr == 1) {
        // printf("Free %lx\n", (long)iptr);
        // Memory is not locked?
        if (*lptr == 0) {            
            // No one holds a lock on it.
            if (dtr) dtr(ptr);
            free(lptr);
        } else {
            *iptr = 0;
        }
    } else {
        // printf("REALLOC %lx %lx\n", (long)*iptr, (long)(INT_MIN + *iptr - 1));
        // References exist. Decrease the reference count and realloc.
        // The remaining memory does not need to be destructed.
        *iptr = INT_MIN + *iptr - 1;        
        // TODO: Use implementation of realloc that ensures that data does not move while shrinking
        if (*lptr == 0) {
            if (dtr) dtr(ptr);
#ifndef VALGRIND
            // No one holds a lock on it.    
            void* ignore = realloc(lptr, 2 * sizeof(int_t));
            assert(ignore == lptr);
#endif
        }
    }
}

void fyr_free_arr(addr_t ptr, fyr_dtr_arr_t dtr) {
    if (ptr == NULL) {
        return;
    }

    // Get locks
    int_t* lptr = ((int_t*)ptr) - 2;
    // Get reference count
    int_t* iptr = ((int_t*)ptr) - 1;
    // Pointer to the allocated area
    void* mem = ((int_t*)ptr) - 3;
    // Only one reference remaining? -> object can be destroyed (unless it is locked)
    if (*iptr == 1) {
        // Memory is not locked?
        if (*lptr == 0) {            
            if (dtr) dtr(ptr, *(((int_t*)ptr) - 3));
            free(mem);
        } else {
            *iptr = 0;
        }
    } else {
        // References exist. Decrease the reference count and realloc.
        // The remaining memory does not need to be destructed.
        *iptr = INT_MIN + *iptr - 1;
        // TODO: Use implementation of realloc that ensures that data does not move while shrinkink
        if (dtr) dtr(ptr, *(((int_t*)ptr) - 3)); 
#ifndef VALGRIND
        void* ignore = realloc(mem, 3*sizeof(int_t));
        assert(ignore == mem);
#endif
    }
}

bool fyr_isnull(addr_t ptr) {
    return ptr == NULL || (*(((int_t*)ptr) - 1) <= 0 && *(((int_t*)ptr) - 2) == 0);
}

void fyr_notnull_ref(addr_t ptr) {
    if (ptr == NULL || (*(((int_t*)ptr) - 1) <= 0 && *(((int_t*)ptr) - 2) == 0)) {
        exit(EXIT_FAILURE);
    }
}

bool fyr_cmp_ref(addr_t ptr1, addr_t ptr2) {
    if (ptr1 == ptr2) {
        return true;
    }
    if (ptr1 == NULL) {
        // Return true, if ptr2 is a weak pointer pointing to a memory area that is destructed
        // and only kept alive because of weak pointers. In this case ptr2 is considered to be NULL as well.
        return ((*(((int_t*)ptr2) - 1) <= 0 && *(((int_t*)ptr2) - 2) == 0));
    } else if (ptr2 == NULL) {
        // Return true, if ptr1 is a weak pointer pointing to a memory area that is destructed
        // and only kept alive because of weak pointers. In this case ptr1 is considered to be NULL as well.
        return ((*(((int_t*)ptr1) - 1) <= 0 && *(((int_t*)ptr1) - 2) == 0));
    }
    return false;
}

addr_t fyr_incref(addr_t ptr) {
    if (ptr == NULL) {
        return NULL;
    }
    int_t* iptr = ((int_t*)ptr) - 1;
    (*iptr)++;
    return ptr;
}

void fyr_decref(addr_t ptr, fyr_dtr_t dtr) {
    if (ptr == NULL) {
        return;
    }
    // Number of locks
    int_t* lptr = ((int_t*)ptr) - 2;
    // Number of references
    int_t* iptr = ((int_t*)ptr) - 1;
    // printf("DECREF %lx\n", (long)*iptr);
    (*iptr)--;
    if (*iptr == 0) {
        // Reference count can drop to zero only when the owning pointer has been assigned
        // to a frozen pointer and all references have been removed.
        // Hence, a destructor must run (unless the objec is locked).
        if (*lptr == 0) {
            if (dtr) dtr(ptr);
            // printf("DECREF FREE %lx\n", (long)iptr);
            free(lptr);
        }
    } else if (*iptr == INT_MIN) {
        // printf("Min count reached\n");
        // The owning pointer is gone, and all references are gone, too.
        // Finally, release all memory, unless someone is holding a lock on the memory
        if (*lptr == 0) {
            // printf("Free refcounter\n");
            // The owning pointer is zero (no freeze) and now all remaining references have been removed.
            // printf("DECREF FREE %lx\n", (long)iptr);
            free(lptr);
        }
    }
}

void fyr_decref_arr(addr_t ptr, fyr_dtr_arr_t dtr) {
    if (ptr == NULL) {
        return;
    }

    // Number of locks
    int_t* lptr = ((int_t*)ptr) - 2;
    // Number of references
    int_t* iptr = ((int_t*)ptr) - 1;
    // Pointer to the allocated area
    void* mem = ((int_t*)ptr) - 3;
    if (--(*iptr) == 0) {
        // Reference count can drop to zero only when the owning pointer has been assigned
        // to a frozen pointer and all references have been removed.
        // Hence, a destructor must run.
        if (*lptr == 0) {
            if (dtr) dtr(ptr, *(((int_t*)ptr) - 3));
            free(mem);
        }
    } else if (*iptr == INT_MIN) {
        // The owning pointer is zero (no freeze) and now all remaining references have been removed.
        free(mem);
    }
}

addr_t fyr_lock(addr_t ptr) {
    if (ptr == NULL) {
        return NULL;
    }
    int_t* lptr = ((int_t*)ptr) - 2;
    (*lptr)++;
    return ptr;
}

void fyr_unlock(addr_t ptr, fyr_dtr_t dtr) {
    if (ptr == NULL) {
        return;
    }
    int_t* lptr = ((int_t*)ptr) - 2;
    int_t* iptr = ((int_t*)ptr) - 1;
    if (--(*lptr) == 0 && *iptr <= 0) {
        if (*iptr == INT_MIN || *iptr == 0) {        
            if (dtr) dtr(ptr);
            free(lptr);
        } else {
            if (dtr) dtr(ptr);
#ifndef VALGRIND
            void* ignore = realloc(lptr, 2 * sizeof(int_t));
            assert(ignore == lptr);
#endif
        }
    }
}

void fyr_unlock_arr(addr_t ptr, fyr_dtr_arr_t dtr) {
    if (ptr == NULL) {
        return;
    }
    int_t* lptr = ((int_t*)ptr) - 2;
    int_t* iptr = ((int_t*)ptr) - 1;
    // Pointer to the allocated area
    void* mem = ((int_t*)ptr) - 3;
    if (--(*lptr) == 0 && *iptr <= 0) {
        if (*iptr == INT_MIN || *iptr == 0) {
            if (dtr) dtr(ptr, *(((int_t*)ptr) - 3));
            free(mem);
        } else {
            if (dtr) dtr(ptr, *(((int_t*)ptr) - 3));
#ifndef VALGRIND
            void* ignore = realloc(mem, 3 * sizeof(int_t));
            assert(ignore == mem);
#endif
        }
    }
}

int_t fyr_len_arr(addr_t ptr) {
    if (ptr == NULL) {
        return 0;
    }
    return *(((int_t*)ptr)-3);
}

int_t fyr_len_str(addr_t ptr) {
    if (ptr == NULL) {
        return 0;
    }
    // -1, because the trailing 0 does not count
    return (*(((int_t*)ptr)-3)) - 1;
}

int_t fyr_min(int_t a, int_t b) {
    if (a < b) {
        return a;
    }
    return b;
}

int_t fyr_max(int_t a, int_t b) {
    if (a > b) {
        return a;
    }
    return b;
}

addr_t fyr_arr_to_str(addr_t array_ptr, addr_t data_ptr, int_t len) {
    if (array_ptr == NULL) {
        return NULL;
    }
    if (array_ptr != data_ptr) {
        memmove(array_ptr, data_ptr, len);
    }
    int *lenptr = ((int_t*)array_ptr)-3;
    // Check for a trailing zero
    if (len >= *lenptr || ((char*)array_ptr)[len] != 0) {
        exit(EXIT_FAILURE);
    }
    *lenptr = len;
    return array_ptr;
}

void fyr_move_arr(addr_t dest, addr_t source, int_t count, int_t size, fyr_dtr_arr_t dtr) {
    if (dest == source) {
        return;
    }
    // Free the destination where it is overwritten by source. Ignore parts that overlap with source
    if (dtr) {
        if (dest < source && dest + count * size > source) {
            dtr(dest, (int)(source - dest)/size);
        } else if (source < dest && source + count * size > dest) {
            dtr(source + count * size, (int)(dest - source)/size);
        } else {
            // No overlap
            dtr(dest, count);            
        }
    }
    memmove(dest, source, count * size);
    if (dest < source && dest + count * size > source) {
        memset(dest + count * size, 0, (int)(source - dest));
    } else if (source < dest && source + count * size > dest) {
        memset(source, 0, (int)(dest - source));
    } else {
        memset(source, 0, count * size);
    }
}
