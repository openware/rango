# RANGO

Rango is a general purpose websocket server which dispatch public and private messages from RabbitMQ.
Rango is made as a drop-in replacement of Ranger built in ruby.
It was designed to be very fast, scalable with a very low memory footprint.

## Configuration

To simplify the migration from Ranger all environement variables are the same:

| VARIABLE          | DEFAULT   | DESCRIPTION                                 |
| ----------------- | --------- | ------------------------------------------- |
| RANGER_HOST       | 0.0.0.0   | Hostname to expose the websocket connection |
| RANGER_PORT       | 8080      | Websocket server port                       |
| RABBITMQ_HOST     | localhost | RabbitMQ hostname to connect to             |
| RABBITMQ_PORT     | 5672      | RabbitMQ port                               |
| RABBITMQ_USER     | guest     | Username used to authenticate to RabbitMQ   |
| RABBITMQ_PASSWORD | guest     | Password used to authenticate to RabbitMQ   |

## Metrics

Rango exposes metrics in Prometheus format on the port 4242.

### Rango metrics

| METRIC                           | TYPE    | DESCRIPTION                                                  |
| -------------------------------- | ------- | ------------------------------------------------------------ |
| rango_hub_clients_count | gauge | Number of clients currently connected |
| rango_hub_subscriptions_count | gauge | Number of user subscribed to a topic |

### HTTP metrics

| METRIC                           | TYPE    | DESCRIPTION                                                  |
| -------------------------------- | ------- | ------------------------------------------------------------ |
| promhttp_metric_handler_requests_in_flight | gauge | Current number of scrapes being served. |
| promhttp_metric_handler_requests_total | counter | Total number of scrapes by HTTP status code. |

### Process metrics

| METRIC                           | TYPE    | DESCRIPTION                                                  |
| -------------------------------- | ------- | ------------------------------------------------------------ |
| process_cpu_seconds_total | counter | Total user and system CPU time spent in seconds. |
| process_max_fds | gauge | Maximum number of open file descriptors. |
| process_open_fds | gauge | Number of open file descriptors. |
| process_resident_memory_bytes | gauge | Resident memory size in bytes. |
| process_start_time_seconds | gauge | Start time of the process since unix epoch in seconds. |
| process_virtual_memory_bytes | gauge | Virtual memory size in bytes. |
| process_virtual_memory_max_bytes | gauge | Maximum amount of virtual memory available in bytes. |

### Go standard metrics

| METRIC                           | TYPE    | DESCRIPTION                                                  |
| -------------------------------- | ------- | ------------------------------------------------------------ |
| go_gc_duration_seconds{quantile} | summary | A summary of the pause duration of garbage collection cycles. |
| go_gc_duration_seconds_sum       |         |                                                              |
| go_gc_duration_seconds_count     |         |                                                              |
| go_goroutines                    | gauge   | Number of goroutines that currently exist.                   |
| go_info{version}                 | gauge   | Information about the Go environment.                        |
| go_memstats_alloc_bytes          | gauge   | Number of bytes allocated and still in use.                  |
| go_memstats_alloc_bytes_total    | counter | Total number of bytes allocated, even if freed.              |
| go_memstats_buck_hash_sys_bytes  | gauge   | Number of bytes used by the profiling bucket hash table.     |
| go_memstats_frees_total          | counter | Total number of frees.                                       |
| go_memstats_gc_cpu_fraction      | gauge   | The fraction of this program's available CPU time used by the GC since the program started. |
| go_memstats_gc_sys_bytes         | gauge   | Number of bytes used for garbage collection system metadata. |
| go_memstats_heap_alloc_bytes     | gauge   | Number of heap bytes allocated and still in use.             |
| go_memstats_heap_idle_bytes      | gauge   | Number of heap bytes waiting to be used.                     |
| go_memstats_heap_inuse_bytes     | gauge   | Number of heap bytes that are in use.                        |
| go_memstats_heap_objects         | gauge   | Number of allocated objects.                                 |
| go_memstats_heap_released_bytes  | gauge   | Number of heap bytes released to OS.                         |
| go_memstats_heap_sys_bytes       | gauge   | Number of heap bytes obtained from system.                   |
| go_memstats_last_gc_time_seconds | gauge   | Number of seconds since 1970 of last garbage collection.     |
| go_memstats_lookups_total        | counter | Total number of pointer lookups.                             |
| go_memstats_mallocs_total        | counter | Total number of mallocs.                                     |
| go_memstats_mcache_inuse_bytes   | gauge   | Number of bytes in use by mcache structures.                 |
| go_memstats_mcache_sys_bytes     | gauge   | Number of bytes used for mcache structures obtained from system. |
| go_memstats_mspan_inuse_bytes    | gauge   | Number of bytes in use by mspan structures.                  |
| go_memstats_mspan_sys_bytes      | gauge   | Number of bytes used for mspan structures obtained from system. |
| go_memstats_next_gc_bytes        | gauge   | Number of heap bytes when next garbage collection will take place. |
| go_memstats_other_sys_bytes      | gauge   | Number of bytes used for other system allocations.           |
| go_memstats_stack_inuse_bytes    | gauge   | Number of bytes in use by the stack allocator.               |
| go_memstats_stack_sys_bytes      | gauge   | Number of bytes obtained from system for stack allocator.    |
| go_memstats_sys_bytes            | gauge   | Number of bytes obtained from system.                        |
| go_threads                       | gauge   | Number of OS threads created.                                |


## Start the server

```bash
./rango
```

## Connect to public channel

```bash
wscat --connect localhost:8080/public
```

## Connect to private channel

```bash
wscat --connect localhost:8080/private --header "Authorization: Bearer ${JWT}"
```

## Messages

### Subscribe to a stream list

```
{"event":"subscribe","streams":["eurusd.trades","eurusd.ob-inc"]}
```

### Unsubscribe to one or several streams

```
{"event":"unsubscribe","streams":["eurusd.trades"]}
```

{"event":"subscribe","streams":["btcusd.trades","ethusd.ob-inc","ethusd.trades","xrpusd.ob-inc","xrpusd.trades","usdtusd.ob-inc","usdtusd.trades"]}
