//+build darwin,macnative

#include "proc_darwin.h"

static const unsigned char info_plist[]
__attribute__ ((section ("__TEXT,__info_plist"),used)) =
"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
"<!DOCTYPE plist PUBLIC \"-//Apple Computer//DTD PLIST 1.0//EN\""
" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n"
"<plist version=\"1.0\">\n"
"<dict>\n"
"  <key>CFBundleIdentifier</key>\n"
"  <string>org.dlv</string>\n"
"  <key>CFBundleName</key>\n"
"  <string>delve</string>\n"
"  <key>CFBundleVersion</key>\n"
"  <string>1.0</string>\n"
"  <key>SecTaskAccess</key>\n"
"  <array>\n"
"    <string>allowed</string>\n"
"    <string>debug</string>\n"
"  </array>\n"
"</dict>\n"
"</plist>\n";

kern_return_t
acquire_mach_task(int tid,
		task_t *task,
		mach_port_t *port_set,
		mach_port_t *exception_port,
		mach_port_t *notification_port)
{
	kern_return_t kret;
	mach_port_t prev_not;
	mach_port_t self = mach_task_self();

	kret = task_for_pid(self, tid, task);
	if (kret != KERN_SUCCESS) return kret;

	// Allocate exception port.
	kret = mach_port_allocate(self, MACH_PORT_RIGHT_RECEIVE, exception_port);
	if (kret != KERN_SUCCESS) return kret;

	kret = mach_port_insert_right(self, *exception_port, *exception_port, MACH_MSG_TYPE_MAKE_SEND);
	if (kret != KERN_SUCCESS) return kret;

	kret = task_set_exception_ports(*task, EXC_MASK_BREAKPOINT|EXC_MASK_SOFTWARE, *exception_port,
			EXCEPTION_DEFAULT, THREAD_STATE_NONE);
	if (kret != KERN_SUCCESS) return kret;

	// Allocate notification port to alert of when the process dies.
	kret = mach_port_allocate(self, MACH_PORT_RIGHT_RECEIVE, notification_port);
	if (kret != KERN_SUCCESS) return kret;

	kret = mach_port_insert_right(self, *notification_port, *notification_port, MACH_MSG_TYPE_MAKE_SEND);
	if (kret != KERN_SUCCESS) return kret;

	kret = mach_port_request_notification(self, *task, MACH_NOTIFY_DEAD_NAME, 0, *notification_port,
			MACH_MSG_TYPE_MAKE_SEND_ONCE, &prev_not);
	if (kret != KERN_SUCCESS) return kret;

	// Create port set.
	kret = mach_port_allocate(self, MACH_PORT_RIGHT_PORT_SET, port_set);
	if (kret != KERN_SUCCESS) return kret;

	// Move exception and notification ports to port set.
	kret = mach_port_move_member(self, *exception_port, *port_set);
	if (kret != KERN_SUCCESS) return kret;

	return mach_port_move_member(self, *notification_port, *port_set);
}

kern_return_t
reset_exception_ports(task_t task, mach_port_t *exception_port, mach_port_t *notification_port) {
	kern_return_t kret;
	mach_port_t prev_not;
	mach_port_t self = mach_task_self();
	
	kret = task_set_exception_ports(task, EXC_MASK_BREAKPOINT|EXC_MASK_SOFTWARE, *exception_port,
			EXCEPTION_DEFAULT, THREAD_STATE_NONE);
	if (kret != KERN_SUCCESS) return kret;
	
	kret = mach_port_request_notification(self, task, MACH_NOTIFY_DEAD_NAME, 0, *notification_port,
			MACH_MSG_TYPE_MAKE_SEND_ONCE, &prev_not);
	if (kret != KERN_SUCCESS) return kret;
	
	return KERN_SUCCESS;
}

char *
find_executable(int pid) {
	static char pathbuf[PATH_MAX];
	proc_pidpath(pid, pathbuf, PATH_MAX);
	return pathbuf;
}

kern_return_t
get_threads(task_t task, void *slice, int limit) {
	kern_return_t kret;
	thread_act_array_t list;
	mach_msg_type_number_t count;

	kret = task_threads(task, &list, &count);
	if (kret != KERN_SUCCESS) {
		return kret;
	}

	if (count > limit) {
		vm_deallocate(mach_task_self(), (vm_address_t) list, count * sizeof(list[0]));
		return -2;
	}

	memcpy(slice, (void*)list, count*sizeof(list[0]));

	kret = vm_deallocate(mach_task_self(), (vm_address_t) list, count * sizeof(list[0]));
	if (kret != KERN_SUCCESS) return kret;

	return (kern_return_t)0;
}

int
thread_count(task_t task) {
	kern_return_t kret;
	thread_act_array_t list;
	mach_msg_type_number_t count;

	kret = task_threads(task, &list, &count);
	if (kret != KERN_SUCCESS) return -1;

	kret = vm_deallocate(mach_task_self(), (vm_address_t) list, count * sizeof(list[0]));
	if (kret != KERN_SUCCESS) return -1;

	return count;
}

mach_port_t
mach_port_wait(mach_port_t port_set, task_t *task, int nonblocking) {
	kern_return_t kret;
	thread_act_t thread;
	NDR_record_t *ndr;
	integer_t *data;
	union
	{
		mach_msg_header_t hdr;
		char data[256];
	} msg;
	mach_msg_option_t opts = MACH_RCV_MSG|MACH_RCV_INTERRUPT;
	if (nonblocking) {
		opts |= MACH_RCV_TIMEOUT;
	}

	// Wait for mach msg.
	kret = mach_msg(&msg.hdr, opts,
			0, sizeof(msg.data), port_set, 10, MACH_PORT_NULL);
	if (kret == MACH_RCV_INTERRUPTED) return kret;
	if (kret != MACH_MSG_SUCCESS) return 0;


	switch (msg.hdr.msgh_id) {
		case 2401: { // Exception
			// 2401 is the exception_raise event, defined in:
			// http://opensource.apple.com/source/xnu/xnu-2422.1.72/osfmk/mach/exc.defs?txt
			// compile this file with mig to get the C version of the description
			
			mach_msg_body_t *bod = (mach_msg_body_t*)(&msg.hdr + 1);
			mach_msg_port_descriptor_t *desc = (mach_msg_port_descriptor_t *)(bod + 1);
			thread = desc[0].name;
			*task = desc[1].name;
			ndr = (NDR_record_t *)(desc + 2);
			data = (integer_t *)(ndr + 1);

			if (thread_suspend(thread) != KERN_SUCCESS) return 0;
			// Send our reply back so the kernel knows this exception has been handled.
			kret = mach_send_reply(msg.hdr);
			if (kret != MACH_MSG_SUCCESS) return 0;
			if (data[2] == EXC_SOFT_SIGNAL) {
				if (data[3] != SIGTRAP) {
					if (thread_resume(thread) != KERN_SUCCESS) return 0;
					return mach_port_wait(port_set, task, nonblocking);
				}
			}
			return thread;
		}

		case 72: { // Death
			// 72 is mach_notify_dead_name, defined in:
			// https://opensource.apple.com/source/xnu/xnu-1228.7.58/osfmk/mach/notify.defs?txt
			// compile this file with mig to get the C version of the description
			ndr = (NDR_record_t *)(&msg.hdr + 1);
			*task = *((mach_port_name_t *)(ndr + 1));
			return msg.hdr.msgh_local_port;
		}
	}
	return 0;
}

kern_return_t
mach_send_reply(mach_msg_header_t hdr) {
	mig_reply_error_t reply;
	mach_msg_header_t *rh = &reply.Head;
	rh->msgh_bits = MACH_MSGH_BITS(MACH_MSGH_BITS_REMOTE(hdr.msgh_bits), 0);
	rh->msgh_remote_port = hdr.msgh_remote_port;
	rh->msgh_size = (mach_msg_size_t) sizeof(mig_reply_error_t);
	rh->msgh_local_port = MACH_PORT_NULL;
	rh->msgh_id = hdr.msgh_id + 100;

	reply.NDR = NDR_record;
	reply.RetCode = KERN_SUCCESS;

	return mach_msg(&reply.Head, MACH_SEND_MSG|MACH_SEND_INTERRUPT, rh->msgh_size, 0,
			MACH_PORT_NULL, MACH_MSG_TIMEOUT_NONE, MACH_PORT_NULL);
}

kern_return_t
raise_exception(mach_port_t task, mach_port_t thread, mach_port_t exception_port, exception_type_t exception) {
	return exception_raise(exception_port, thread, task, exception, 0, 0);
}

task_t
get_task_for_pid(int pid) {
	task_t task = 0;
	mach_port_t self = mach_task_self();

	task_for_pid(self, pid, &task);
	return task;
}

int
task_is_valid(task_t task) {
	struct task_basic_info info;
	mach_msg_type_number_t count = TASK_BASIC_INFO_COUNT;
	return task_info(task, TASK_BASIC_INFO, (task_info_t)&info, &count) == KERN_SUCCESS;
}
