#ifndef GOKVM_AVC_NATIVE_H
#define GOKVM_AVC_NATIVE_H

typedef struct gokvm_avc_encoder gokvm_avc_encoder;
int gokvm_avc_create(int width, int height, gokvm_avc_encoder **out);
int gokvm_avc_encode(gokvm_avc_encoder *encoder, const unsigned char *i420,
                     int force, long long timestamp, unsigned char **data, int *size);
void gokvm_avc_destroy(gokvm_avc_encoder *encoder);

#endif
