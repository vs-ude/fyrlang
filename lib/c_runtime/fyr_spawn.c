#include <stdlib.h>
#include <stdint.h>
#include <alloca.h>

#include "fyr.h"
#include "fyr_spawn.h"

//#include <stdio.h>

struct fyr_coro_t fyr_main_coro;
struct fyr_coro_t *fyr_running;
struct fyr_coro_t *fyr_ready_first;
struct fyr_coro_t *fyr_ready_last;
struct fyr_coro_t *fyr_ready2_first;
struct fyr_coro_t *fyr_ready2_last;
struct fyr_coro_t *fyr_waiting;
struct fyr_coro_t *fyr_garbage_coro;

#define fyr_coro_STACKSIZE (16*1024)

void fyr_component_main_start(void) {
    fyr_main_coro.memory = NULL;
    fyr_main_coro.next = NULL;
    fyr_running = &fyr_main_coro;
    fyr_ready_first = NULL;
    fyr_ready_last = NULL;
    fyr_ready2_first = NULL;
    fyr_ready2_last = NULL;
    fyr_waiting = NULL;
    fyr_garbage_coro = NULL;
}

void fyr_component_main_end(void) {
    // The main coroutine has finished.
    fyr_running = NULL;
    // Set a jump such that fyr_yield can return to this point.
    if (!setjmp(fyr_main_coro.buf)) {
        // If there are other coroutines waiting to be executed, then execute them.
        // This yield will not return. It will jump to the position we set before.
        fyr_yield(true);
    }
    // We are here, because fyr_yield decided that there are no more coroutines left.
    //
    // The previous coroutine finished? Garbage collect it now.
    // Doing that before was not possible, because a coroutine cannot delete the stack it operates on.
    if (fyr_garbage_coro) {
//        printf("Free ...\n");
        fyr_free(fyr_garbage_coro->memory, NULL);
        fyr_garbage_coro = NULL;
    }
}

void fyr_yield(bool wait) {
//    printf("yield ... %p\n", fyr_running);
    if (fyr_running) {
        if (setjmp(fyr_running->buf)) {
            // The previous coroutine finished? Garbage collect it now.
            // Doing that before was not possible because a coroutine cannot delete the stack it operates on.
            if (fyr_garbage_coro) {
//                printf("Free ...\n");
                fyr_free(fyr_garbage_coro->memory, NULL);
                fyr_garbage_coro = NULL;
            }
            // When we are here, the yielding coroutine is resumed.
            return;
        }
    }
    if (fyr_ready_first == NULL && fyr_ready2_first == NULL) {
        // All other coroutines are waiting to be resumed, only the yielding coroutine can continue?
        // Then continue the yielding coroutine.
        if (!wait) {
            return;
        }
        if (fyr_waiting != NULL || fyr_running != NULL) {
            // There are coroutines left, but all are waiting. This is a deadlock.
            exit(1);
        }
        // There are no coroutines left.
        // This implies that the main coroutine must have completed and fyr_component_main_end has been called.
        // Jump there.
        longjmp(fyr_main_coro.buf, 1);
    }
    // Put the current co-routine in the waiting or ready list.
    // Do nothing like that if the current coroutine has finished (i.e. fyr_running == NULL).
    if (fyr_running) {
        if (wait) {
            // Add the current coroutine to the waiting list
            fyr_running->next = fyr_waiting;
            if (fyr_waiting != NULL) {
                fyr_waiting->prev = fyr_running;
            }
            fyr_waiting = fyr_running;
        } else {
            // Add the current coroutine at the end of the ready2 list.
            if (fyr_ready2_first == NULL) {
                fyr_ready2_first = fyr_running;
                fyr_ready2_last = fyr_running;
            } else {
                fyr_ready2_last->next = fyr_running;
                fyr_ready2_last = fyr_running;
            }
        }
    }
    // Execute the next coroutine that is ready.
    // Look in both ready lists.
    if (fyr_ready_first != NULL) {
        fyr_running = fyr_ready_first;
        if (fyr_ready_first == fyr_ready_last) {
            // The ready list is now empty
            fyr_ready_first = NULL;
            fyr_ready_last = NULL;
        } else {
            fyr_ready_first = fyr_ready_first->next;
        }
    } else {
        fyr_running = fyr_ready2_first;
        if (fyr_ready2_first == fyr_ready2_last) {
            // The ready2 list is now empty
            fyr_ready2_first = NULL;
            fyr_ready2_last = NULL;
        } else {
            fyr_ready2_first = fyr_ready2_first->next;
        }
    }
//    printf("CORO running %p, main is %p\n", fyr_running, &fyr_main_coro);
    fyr_running->next = NULL;
//    printf("Jumping ...\n");
    longjmp(fyr_running->buf, 1);    
}

int fyr_stacksize() {
    return fyr_coro_STACKSIZE + sizeof(struct fyr_coro_t);
}

void fyr_resume(struct fyr_coro_t *c) {
    if (fyr_waiting == c) {
        // The coroutine is the first element of the waiting list?
        fyr_waiting = c->next;
        if (c->next != NULL) {
            c->next->prev = NULL;
            c->next = NULL;
        }    
    } else if (c->prev != NULL) {
        // The coroutine is in a double linked list? This must be the waiting list.
        c->prev->next = c->next;
        if (c->next != NULL) {
            c->next->prev = c->prev;
            c->next = NULL;
        }    
        c->prev = NULL;
    } else {
        // The coroutine is not in the waiting list. Do nothing.
        return;
    }
    // Add the coroutine to the end of the ready list
    if (fyr_ready_last == NULL) {
        fyr_ready_last = c;
        fyr_ready_first = c;
    } else {
        fyr_ready_last->next = c;
        fyr_ready2_last = c;
    }
}

struct fyr_coro_t* fyr_coroutine(void) {
    fyr_incref((addr_t)fyr_running);
    return fyr_running;
}
