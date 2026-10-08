// Test-only OpenH264 decoder. Input is a sequence of length-prefixed access
// units; output is width, height, then packed I420 samples for each frame.
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <wels/codec_api.h>

static void write_u32(unsigned value) {
    unsigned char bytes[] = {value, value >> 8, value >> 16, value >> 24};
    fwrite(bytes, 1, 4, stdout);
}

int main(void) {
    ISVCDecoder *decoder = NULL;
    if (WelsCreateDecoder(&decoder) || !decoder) return 1;
    SDecodingParam params;
    memset(&params, 0, sizeof(params));
    params.sVideoProperty.eVideoBsType = VIDEO_BITSTREAM_AVC;
    if ((*decoder)->Initialize(decoder, &params)) return 2;
    unsigned char prefix[4];
    while (fread(prefix, 1, 4, stdin) == 4) {
        unsigned length = prefix[0] | prefix[1] << 8 | prefix[2] << 16 | (unsigned)prefix[3] << 24;
        if (!length || length > 16 * 1024 * 1024) return 3;
        unsigned char *input = malloc(length);
        if (!input || fread(input, 1, length, stdin) != length) return 4;
        SBufferInfo info;
        memset(&info, 0, sizeof(info));
        unsigned char *planes[3] = {NULL};
        DECODING_STATE status = (*decoder)->DecodeFrameNoDelay(decoder, input, length, planes, &info);
        free(input);
        if (status || info.iBufferStatus != 1) return 5;
        SSysMEMBuffer *buffer = &info.UsrData.sSystemBuffer;
        write_u32(buffer->iWidth);
        write_u32(buffer->iHeight);
        for (int p = 0; p < 3; p++) {
            int width = buffer->iWidth / (p ? 2 : 1);
            int height = buffer->iHeight / (p ? 2 : 1);
            int stride = buffer->iStride[p ? 1 : 0];
            for (int y = 0; y < height; y++) fwrite(planes[p] + y * stride, 1, width, stdout);
        }
    }
    (*decoder)->Uninitialize(decoder);
    WelsDestroyDecoder(decoder);
    return 0;
}
