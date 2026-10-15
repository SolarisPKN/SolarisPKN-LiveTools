package com.solarispkn.movilcam;

import java.io.ByteArrayOutputStream;

/**
 * OBS/FFmpeg expects Annex-B H.264 start codes on the TCP byte stream.
 * Some hardware encoders return Annex-B, others return 4-byte AVCC lengths.
 * Keep this code Android-free so it is testable on a workstation.
 */
public final class H264AnnexB {
    private static final byte[] PREFIX = {0, 0, 0, 1};

    private H264AnnexB() {}

    public static boolean containsStartCode(byte[] data) {
        // AVCC length prefixes elsewhere in the packet may coincidentally equal 00 00 00 01.
        // Annex-B must begin with a start code; checking the entire packet would misclassify AVCC.
        return data != null && (
            (data.length >= 3 && data[0] == 0 && data[1] == 0 && data[2] == 1) ||
            (data.length >= 4 && data[0] == 0 && data[1] == 0 &&
             data[2] == 0 && data[3] == 1));
    }

    public static byte[] normalize(byte[] data) {
        if (data == null || data.length == 0) return new byte[0];
        if (containsStartCode(data)) return data;

        ByteArrayOutputStream converted = new ByteArrayOutputStream(data.length + 16);
        int off = 0;
        boolean parsed = false;
        while (off + 4 < data.length) {
            long length = ((data[off] & 0xffL) << 24)
                | ((data[off + 1] & 0xffL) << 16)
                | ((data[off + 2] & 0xffL) << 8)
                | (data[off + 3] & 0xffL);
            if (length <= 0 || length > data.length - off - 4) {
                parsed = false;
                break;
            }
            converted.write(PREFIX, 0, PREFIX.length);
            converted.write(data, off + 4, (int) length);
            off += 4 + (int) length;
            parsed = true;
        }
        if (parsed && off == data.length) return converted.toByteArray();

        // Raw single NAL unit (common for codec-specific data).
        ByteArrayOutputStream one = new ByteArrayOutputStream(data.length + 4);
        one.write(PREFIX, 0, PREFIX.length);
        one.write(data, 0, data.length);
        return one.toByteArray();
    }

    public static byte[] config(byte[] sps, byte[] pps) {
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        for (byte[] unit : new byte[][]{sps, pps}) {
            byte[] annex = normalize(unit);
            out.write(annex, 0, annex.length);
        }
        return out.toByteArray();
    }
}
