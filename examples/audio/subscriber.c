#include "libsoundio/soundio/soundio.h"
#include "clivekit.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <stdarg.h>

struct RecordContext {
    struct SoundIoRingBuffer *ring_buffer;
};

static enum SoundIoFormat prioritized_formats[] = {
    SoundIoFormatFloat32NE,
    SoundIoFormatFloat32FE,
    SoundIoFormatS32NE,
    SoundIoFormatS32FE,
    SoundIoFormatS24NE,
    SoundIoFormatS24FE,
    SoundIoFormatS16NE,
    SoundIoFormatS16FE,
    SoundIoFormatFloat64NE,
    SoundIoFormatFloat64FE,
    SoundIoFormatU32NE,
    SoundIoFormatU32FE,
    SoundIoFormatU24NE,
    SoundIoFormatU24FE,
    SoundIoFormatU16NE,
    SoundIoFormatU16FE,
    SoundIoFormatS8,
    SoundIoFormatU8,
    SoundIoFormatInvalid,
};

static int prioritized_sample_rates[] = {
    48000,
    44100,
    96000,
    24000,
    0,
};

__attribute__ ((cold))
__attribute__ ((noreturn))
__attribute__ ((format (printf, 1, 2)))
static void panic(const char *format, ...) {
    va_list ap;
    va_start(ap, format);
    vfprintf(stderr, format, ap);
    fprintf(stderr, "\n");
    va_end(ap);
    abort();
}

static int min_int(int a, int b) {
    return (a < b) ? a : b;
}

static void write_callback(struct SoundIoOutStream *outstream, int frame_count_min, int frame_count_max) {
    struct RecordContext *rc = outstream->userdata;

    struct SoundIoChannelArea *areas;
    int frames_left;
    int frame_count;
    int err;

    char *read_ptr = soundio_ring_buffer_read_ptr(rc->ring_buffer);
    int fill_bytes = soundio_ring_buffer_fill_count(rc->ring_buffer);
    int fill_count = fill_bytes / outstream->bytes_per_frame;

    if (frame_count_min > fill_count) {
        // Ring buffer does not have enough data, fill with zeroes.
        frames_left = frame_count_min;
        for (;;) {
            frame_count = frames_left;
            if (frame_count <= 0)
              return;
            if ((err = soundio_outstream_begin_write(outstream, &areas, &frame_count)))
                panic("begin write error: %s", soundio_strerror(err));
            if (frame_count <= 0)
                return;
            for (int frame = 0; frame < frame_count; frame += 1) {
                for (int ch = 0; ch < outstream->layout.channel_count; ch += 1) {
                    memset(areas[ch].ptr, 0, outstream->bytes_per_sample);
                    areas[ch].ptr += areas[ch].step;
                }
            }
            if ((err = soundio_outstream_end_write(outstream)))
                panic("end write error: %s", soundio_strerror(err));
            frames_left -= frame_count;
        }
    }

    int read_count = min_int(frame_count_max, fill_count);
    frames_left = read_count;

    while (frames_left > 0) {
        int frame_count = frames_left;

        if ((err = soundio_outstream_begin_write(outstream, &areas, &frame_count)))
            panic("begin write error: %s", soundio_strerror(err));

        if (frame_count <= 0)
            break;

        for (int frame = 0; frame < frame_count; frame += 1) {
            for (int ch = 0; ch < outstream->layout.channel_count; ch += 1) {
                memcpy(areas[ch].ptr, read_ptr, outstream->bytes_per_sample);
                areas[ch].ptr += areas[ch].step;
                read_ptr += outstream->bytes_per_sample;
            }
        }

        if ((err = soundio_outstream_end_write(outstream)))
            panic("end write error: %s", soundio_strerror(err));

        frames_left -= frame_count;
    }

    soundio_ring_buffer_advance_read_ptr(rc->ring_buffer, read_count * outstream->bytes_per_frame);
}

static void underflow_callback(struct SoundIoOutStream *outstream) {
    struct RecordContext *rc = outstream->userdata;
    int capacity_row = outstream->sample_rate * outstream->bytes_per_frame;
    soundio_ring_buffer_clear(rc->ring_buffer);

    char *buf = soundio_ring_buffer_write_ptr(rc->ring_buffer);
    memset(buf, 0, capacity_row);
    soundio_ring_buffer_advance_write_ptr(rc->ring_buffer, capacity_row);
}

static int usage(char *exe) {
    fprintf(stderr, "Usage: %s [options] outfile\n"
            "Options:\n"
            "  [--backend dummy|alsa|pulseaudio|jack|coreaudio|wasapi]\n"
            "  [--device id]\n"
            "  [--raw]\n"
            , exe);
    return 1;
}

int main(int argc, char **argv) {
    char *exe = argv[0];
    enum SoundIoBackend backend = SoundIoBackendNone;
    char *device_id = NULL;
    bool is_raw = false;
    char *infile = NULL;
    for (int i = 1; i < argc; i += 1) {
        char *arg = argv[i];
        if (arg[0] == '-' && arg[1] == '-') {
            if (strcmp(arg, "--raw") == 0) {
                is_raw = true;
            } else if (++i >= argc) {
                return usage(exe);
            } else if (strcmp(arg, "--backend") == 0) {
                if (strcmp("dummy", argv[i]) == 0) {
                    backend = SoundIoBackendDummy;
                } else if (strcmp("alsa", argv[i]) == 0) {
                    backend = SoundIoBackendAlsa;
                } else if (strcmp("pulseaudio", argv[i]) == 0) {
                    backend = SoundIoBackendPulseAudio;
                } else if (strcmp("jack", argv[i]) == 0) {
                    backend = SoundIoBackendJack;
                } else if (strcmp("coreaudio", argv[i]) == 0) {
                    backend = SoundIoBackendCoreAudio;
                } else if (strcmp("wasapi", argv[i]) == 0) {
                    backend = SoundIoBackendWasapi;
                } else {
                    fprintf(stderr, "Invalid backend: %s\n", argv[i]);
                    return 1;
                }
            } else if (strcmp(arg, "--device") == 0) {
                device_id = argv[i];
            } else {
                return usage(exe);
            }
        } else if (!infile) {
            infile = argv[i];
        } else {
            return usage(exe);
        }
    }

    if (!infile)
        return usage(exe);

    struct RecordContext rc;

    struct SoundIo *soundio = soundio_create();
    if (!soundio) {
        fprintf(stderr, "out of memory\n");
        return 1;
    }

    int err = (backend == SoundIoBackendNone) ?
        soundio_connect(soundio) : soundio_connect_backend(soundio, backend);
    if (err) {
        fprintf(stderr, "error connecting: %s", soundio_strerror(err));
        return 1;
    }

    soundio_flush_events(soundio);

    struct SoundIoDevice *selected_device = NULL;

    if (device_id) {
        for (int i = 0; i < soundio_output_device_count(soundio); i += 1) {
            struct SoundIoDevice *device = soundio_get_output_device(soundio, i);
            if (device->is_raw == is_raw && strcmp(device->id, device_id) == 0) {
                selected_device = device;
                break;
            }
            soundio_device_unref(device);
        }
        if (!selected_device) {
            fprintf(stderr, "Invalid device id: %s\n", device_id);
            return 1;
        }
    } else {
        int device_index = soundio_default_output_device_index(soundio);
        selected_device = soundio_get_output_device(soundio, device_index);
        if (!selected_device) {
            fprintf(stderr, "No output devices available.\n");
            return 1;
        }
    }

    fprintf(stderr, "Device: %s\n", selected_device->name);

    if (selected_device->probe_error) {
        fprintf(stderr, "Unable to probe device: %s\n", soundio_strerror(selected_device->probe_error));
        return 1;
    }

    soundio_device_sort_channel_layouts(selected_device);

    int sample_rate = 0;
    int *sample_rate_ptr;
    for (sample_rate_ptr = prioritized_sample_rates; *sample_rate_ptr; sample_rate_ptr += 1) {
        if (soundio_device_supports_sample_rate(selected_device, *sample_rate_ptr)) {
            sample_rate = *sample_rate_ptr;
            break;
        }
    }
    if (!sample_rate)
        sample_rate = selected_device->sample_rates[0].max;

    enum SoundIoFormat fmt = SoundIoFormatInvalid;
    enum SoundIoFormat *fmt_ptr;
    for (fmt_ptr = prioritized_formats; *fmt_ptr != SoundIoFormatInvalid; fmt_ptr += 1) {
        if (soundio_device_supports_format(selected_device, *fmt_ptr)) {
            fmt = *fmt_ptr;
            break;
        }
    }
    if (fmt == SoundIoFormatInvalid)
        fmt = selected_device->formats[0];

    struct SoundIoOutStream *outstream = soundio_outstream_create(selected_device);
    if (!outstream) {
        fprintf(stderr, "out of memory\n");
        return 1;
    }
    outstream->format = fmt;
    outstream->sample_rate = sample_rate;
    outstream->write_callback = write_callback;
    outstream->userdata = &rc;
    outstream->software_latency = 0.2;
    outstream->underflow_callback = underflow_callback;

    if ((err = soundio_outstream_open(outstream))) {
        fprintf(stderr, "unable to open input stream: %s", soundio_strerror(err));
        return 1;
    }

    fprintf(stderr, "%s %dHz %s interleaved\n",
            outstream->layout.name, sample_rate, soundio_format_string(fmt));

    const int ring_buffer_duration_seconds = 5;
    int capacity_row = outstream->sample_rate * outstream->bytes_per_frame;
    int capacity = ring_buffer_duration_seconds * capacity_row;

    rc.ring_buffer = soundio_ring_buffer_create(soundio, capacity);
    if (!rc.ring_buffer) {
        fprintf(stderr, "out of memory\n");
        return 1;
    }

    char *buf = soundio_ring_buffer_write_ptr(rc.ring_buffer);
    memset(buf, 0, capacity_row);
    soundio_ring_buffer_advance_write_ptr(rc.ring_buffer, capacity_row);
    
    if ((err = soundio_outstream_start(outstream))) {
        fprintf(stderr, "unable to start input device: %s", soundio_strerror(err));
        return 1;
    }

    char room_desc[CLIVEKIT_SIZE_DESC];

    clivekit_connect_info conn_info = {
        .host = "ws://localhost:7880",
        .api_key = "devkey",
        .api_secret = "secret",
        .room_name = "test",
        .ident = "subscriber"
    };

    int status = clivekit_connect_to_room(room_desc, conn_info);
    if (status) {
        printf("connect failed\n");
        return 1;
    }

    printf("connect success\n");

    char rx_key[CLIVEKIT_SIZE_ENCKEY] = {0};
    status = clivekit_add_rx_key_for_room(room_desc, "publisher", rx_key);
    if (status) {
        printf("set rx_key\n");
        return 2;
    }

    for (;;) {
        int fill_count = soundio_ring_buffer_fill_count(rc.ring_buffer);
        char *write_buf = soundio_ring_buffer_write_ptr(rc.ring_buffer);

        if (fill_count >= capacity_row) {
            continue;
        }

        clivekit_data_packet data_packet;
        int status = clivekit_read_data_from_room(room_desc, &data_packet);
        if (status) {
            printf("read failed\n");
            return 3;
        }

        printf("READ %d - %zu\n", data_packet.dtype, data_packet.payload_size);

        memcpy(write_buf, data_packet.payload, data_packet.payload_size);
        soundio_ring_buffer_advance_write_ptr(rc.ring_buffer, data_packet.payload_size);
    }

    clivekit_disconnect_from_room(room_desc);

    soundio_outstream_destroy(outstream);
    soundio_device_unref(selected_device);
    soundio_destroy(soundio);
    return 0;
}
