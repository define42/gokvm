//go:build openh264 && cgo

#include "native.h"
#include <stdlib.h>
#include <string.h>
#include <wels/codec_api.h>
#include <wels/codec_ver.h>

enum { GOKVM_AVC_MAX_FRAME = 16 * 1024 * 1024 };

struct gokvm_avc_encoder {
    ISVCEncoder *codec;
    unsigned char *i420;
    unsigned char *output;
    int capacity;
    int width, height;
};

void gokvm_avc_destroy(gokvm_avc_encoder *encoder) {
    if (!encoder) return;
    if (encoder->codec) {
        (*encoder->codec)->Uninitialize(encoder->codec);
        WelsDestroySVCEncoder(encoder->codec);
    }
    free(encoder->i420);
    free(encoder->output);
    free(encoder);
}

int gokvm_avc_create(int width, int height, gokvm_avc_encoder **out) {
    OpenH264Version version;
    WelsGetCodecVersionEx(&version);
    // Extended encoder structures can change across OpenH264 releases.
    if (version.uMajor != OPENH264_MAJOR || version.uMinor != OPENH264_MINOR) return -1;

    gokvm_avc_encoder *encoder = calloc(1, sizeof(*encoder));
    if (!encoder) return -2;
    encoder->width = width;
    encoder->height = height;
    encoder->i420 = malloc((size_t)width * height * 3 / 2);
    int status = WelsCreateSVCEncoder(&encoder->codec);
    if (status || !encoder->codec || !encoder->i420) {
        gokvm_avc_destroy(encoder);
        return status ? status : -2;
    }

    SEncParamExt params;
    memset(&params, 0, sizeof(params));
    status = (*encoder->codec)->GetDefaultParams(encoder->codec, &params);
    if (status) {
        gokvm_avc_destroy(encoder);
        return status;
    }
    params.iUsageType = SCREEN_CONTENT_REAL_TIME;
    params.iPicWidth = width;
    params.iPicHeight = height;
    params.fMaxFrameRate = 60;
    params.iRCMode = RC_OFF_MODE;
    params.iSpatialLayerNum = 1;
    params.iTemporalLayerNum = 1;
    params.bSimulcastAVC = false;
    params.iMultipleThreadIdc = 1;
    params.uiIntraPeriod = 150;
    params.bEnableFrameSkip = false;
    params.bEnableDenoise = false;
    params.bEnableAdaptiveQuant = false;
    params.bEnableBackgroundDetection = false;
    params.bEnableLongTermReference = false;
    params.bPrefixNalAddingCtrl = false;
    params.bEnableSSEI = false;
    params.eSpsPpsIdStrategy = CONSTANT_ID;
    SSpatialLayerConfig *layer = &params.sSpatialLayers[0];
    layer->iVideoWidth = width;
    layer->iVideoHeight = height;
    layer->fFrameRate = 60;
    layer->uiProfileIdc = PRO_BASELINE;
    layer->iDLayerQp = 20;
    layer->sSliceArgument.uiSliceMode = SM_SINGLE_SLICE;
    layer->bVideoSignalTypePresent = true;
    layer->uiVideoFormat = VF_UNDEF;
    layer->bFullRange = true;
    layer->bColorDescriptionPresent = true;
    layer->uiColorPrimaries = CP_BT709;
    layer->uiTransferCharacteristics = TRC_BT709;
    layer->uiColorMatrix = CM_BT709;
    status = (*encoder->codec)->InitializeExt(encoder->codec, &params);
    if (status) {
        gokvm_avc_destroy(encoder);
        return status;
    }
    *out = encoder;
    return 0;
}

int gokvm_avc_encode(gokvm_avc_encoder *encoder, const unsigned char *i420,
                     int force, long long timestamp, unsigned char **data, int *size) {
    int status;
    if (force) {
        status = (*encoder->codec)->ForceIntraFrame(encoder->codec, true);
        if (status) return status;
    }
    const int luma = encoder->width * encoder->height;
    memcpy(encoder->i420, i420, (size_t)luma * 3 / 2);
    SSourcePicture picture;
    memset(&picture, 0, sizeof(picture));
    picture.iColorFormat = videoFormatI420;
    picture.iPicWidth = picture.iStride[0] = encoder->width;
    picture.iPicHeight = encoder->height;
    picture.iStride[1] = picture.iStride[2] = encoder->width / 2;
    picture.pData[0] = encoder->i420;
    picture.pData[1] = encoder->i420 + luma;
    picture.pData[2] = encoder->i420 + luma + luma / 4;
    picture.uiTimeStamp = timestamp;
    SFrameBSInfo frame;
    memset(&frame, 0, sizeof(frame));
    status = (*encoder->codec)->EncodeFrame(encoder->codec, &picture, &frame);
    if (status) return status;
    if (frame.iLayerNum < 1 || frame.iLayerNum > MAX_LAYER_NUM_OF_FRAME) return -3;

    int length = 0;
    for (int i = 0; i < frame.iLayerNum; ++i) {
        const SLayerBSInfo *layer = &frame.sLayerInfo[i];
        if (layer->iNalCount < 0 || !layer->pNalLengthInByte || !layer->pBsBuf) return -3;
        for (int j = 0; j < layer->iNalCount; ++j) {
            const int n = layer->pNalLengthInByte[j];
            if (n < 1 || n > GOKVM_AVC_MAX_FRAME - length) return -3;
            length += n;
        }
    }
    if (length == 0) return -3;
    if (length > encoder->capacity) {
        unsigned char *output = realloc(encoder->output, (size_t)length);
        if (!output) return -2;
        encoder->output = output;
        encoder->capacity = length;
    }
    int offset = 0;
    for (int i = 0; i < frame.iLayerNum; ++i) {
        const SLayerBSInfo *layer = &frame.sLayerInfo[i];
        int bytes = 0;
        for (int j = 0; j < layer->iNalCount; ++j) bytes += layer->pNalLengthInByte[j];
        memcpy(encoder->output + offset, layer->pBsBuf, (size_t)bytes);
        offset += bytes;
    }
    *data = encoder->output;
    *size = length;
    return 0;
}
