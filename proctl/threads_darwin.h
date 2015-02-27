#include <stdio.h>
#include <sys/types.h>
#include <mach/mach.h>
#include <mach/mach_vm.h>
#include <mach/thread_info.h>

int
write_memory(mach_port_name_t, mach_vm_address_t, void *, mach_msg_type_number_t);

int
read_memory(mach_port_name_t, mach_vm_address_t, void *, mach_msg_type_number_t);

x86_thread_state64_t
get_registers(mach_port_name_t);

void
set_pc(thread_act_t, uint64_t);

void
single_step(thread_act_t);

void
clear_trap_flag(thread_act_t thread);
