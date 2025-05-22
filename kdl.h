#include <kdl/kdl.h>

typedef char const *kdlgo_char_const_ptr;

size_t kdlgo_read(void *ptr, char *buf, size_t bufsize);
size_t kdlgo_write(void *ptr, char const *data, size_t nbytes);
