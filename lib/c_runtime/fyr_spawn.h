#ifndef FYR_SPAWN
#define FYR_SPAWN

#include <stdbool.h>
#include <setjmp.h>

struct fyr_coro_t {
    void* memory;
    struct fyr_coro_t *prev;
    struct fyr_coro_t *next;
    jmp_buf buf;
};

extern struct fyr_coro_t fyr_main_coro;
extern struct fyr_coro_t *fyr_running;
extern struct fyr_coro_t *fyr_ready_first;
extern struct fyr_coro_t *fyr_ready_last;
extern struct fyr_coro_t *fyr_waiting;
extern struct fyr_coro_t *fyr_garbage_coro;

void fyr_component_main_start(void);
void fyr_component_main_end(void);
void fyr_yield(bool);
int fyr_stacksize();
void fyr_resume(struct fyr_coro_t *coro);
struct fyr_coro_t* fyr_coroutine(void);

#endif