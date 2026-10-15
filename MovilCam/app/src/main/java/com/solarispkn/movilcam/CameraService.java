package com.solarispkn.movilcam;

import android.Manifest;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.ServiceInfo;
import android.graphics.ImageFormat;
import android.hardware.camera2.CameraAccessException;
import android.hardware.camera2.CameraCaptureSession;
import android.hardware.camera2.CameraCharacteristics;
import android.hardware.camera2.CameraDevice;
import android.hardware.camera2.CameraManager;
import android.hardware.camera2.CaptureRequest;
import android.hardware.camera2.params.StreamConfigurationMap;
import android.media.MediaCodec;
import android.media.MediaCodecInfo;
import android.media.MediaCodecList;
import android.media.MediaFormat;
import android.media.MediaRecorder;
import android.os.Handler;
import android.os.HandlerThread;
import android.os.IBinder;
import android.util.Log;
import android.util.Range;
import android.util.Size;
import android.view.Surface;

import java.io.BufferedOutputStream;
import java.io.Closeable;
import java.io.IOException;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.ByteBuffer;
import java.util.Arrays;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Foreground service; camera and encoder are used ONLY while a USB/ADB client is connected.
 * Pure H.264 Annex-B TCP stream, 127.0.0.1:27183. No WebView, screen preview or microphone.
 */
public final class CameraService extends Service {
    private static final String TAG = "MovilCam";
    private static final String CHANNEL = "movilcam_capture";
    private static final int NOTIFICATION_ID = 113;
    public static final int PORT = 27183;

    private volatile boolean running;
    private volatile Options options = new Options("back", 1920, 1080, 30, 8_000_000);
    private volatile ServerSocket listener;
    private volatile Socket currentClient;
    private Thread worker;

    static final class Options {
        final String lens;
        final int width, height, fps, bitrate;
        Options(String lens, int width, int height, int fps, int bitrate) {
            this.lens = lens;
            this.width = width;
            this.height = height;
            this.fps = fps;
            this.bitrate = bitrate;
        }
    }

    @Override public IBinder onBind(Intent intent) { return null; }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        if (checkSelfPermission(Manifest.permission.CAMERA) != PackageManager.PERMISSION_GRANTED) {
            Log.e(TAG, "Camera permission missing. adb shell pm grant com.solarispkn.movilcam android.permission.CAMERA");
            stopSelf();
            return START_NOT_STICKY;
        }

        if (intent != null) {
            String lens = intent.getStringExtra("lens");
            options = new Options(
                "front".equalsIgnoreCase(lens) ? "front" : "back",
                clamp(intent.getIntExtra("width", 1920), 320, 3840),
                clamp(intent.getIntExtra("height", 1080), 240, 2160),
                clamp(intent.getIntExtra("fps", 30), 5, 60),
                clamp(intent.getIntExtra("bitrate", 8_000_000), 500_000, 24_000_000));
        }

        NotificationManager manager = getSystemService(NotificationManager.class);
        manager.createNotificationChannel(new NotificationChannel(CHANNEL,
            "MovilCam camera transport", NotificationManager.IMPORTANCE_LOW));
        Notification notice = new Notification.Builder(this, CHANNEL)
            .setContentTitle("SolarisPKN MovilCam")
            .setContentText("USB camera server; capture only while connected")
            .setSmallIcon(android.R.drawable.ic_menu_camera)
            .setOngoing(true)
            .build();
        startForeground(NOTIFICATION_ID, notice, ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA);

        if (!running) {
            running = true;
            worker = new Thread(this::serve, "MovilCam-transport");
            worker.start();
        } else {
            // Force reconnect with new lens/settings; no stale codec remains active.
            closeQuietly(currentClient);
        }
        return START_NOT_STICKY;
    }

    private static int clamp(int value, int low, int high) {
        return Math.max(low, Math.min(high, value));
    }

    private void serve() {
        try (ServerSocket server = new ServerSocket()) {
            listener = server;
            server.setReuseAddress(true);
            server.bind(new InetSocketAddress(InetAddress.getByName("127.0.0.1"), PORT), 1);
            server.setSoTimeout(1000);
            Log.i(TAG, "Listening on device loopback:" + PORT + ". Awaiting ADB forwarded OBS client.");
            while (running) {
                try {
                    Socket socket = server.accept();
                    currentClient = socket;
                    socket.setTcpNoDelay(true);
                    socket.setSendBufferSize(256 * 1024);
                    Log.i(TAG, "OBS client connected; starting camera.");
                    try (Socket client = socket;
                         BufferedOutputStream out =
                             new BufferedOutputStream(client.getOutputStream(), 64 * 1024)) {
                        runCamera(options, out);
                    } catch (Exception e) {
                        if (running) Log.e(TAG, "Capture stopped: " + e);
                    } finally {
                        currentClient = null;
                        Log.i(TAG, "Client disconnected; releasing camera and encoder.");
                    }
                } catch (java.net.SocketTimeoutException ignored) {
                    // Poll running flag while idle. No camera is active.
                } catch (java.net.SocketException e) {
                    if (running) Log.e(TAG, "TCP error", e);
                }
            }
        } catch (Exception e) {
            if (running) Log.e(TAG, "USB transport failed", e);
        } finally {
            listener = null;
            if (running) stopSelf();
        }
    }

    private void runCamera(Options cfg, BufferedOutputStream destination) throws Exception {
        HandlerThread cameraThread = new HandlerThread("MovilCam-Camera2");
        cameraThread.start();
        Handler cameraHandler = new Handler(cameraThread.getLooper());
        MediaCodec encoder = null;
        Surface inputSurface = null;
        CameraDevice device = null;
        CameraCaptureSession session = null;
        try {
            CameraManager manager = (CameraManager) getSystemService(CAMERA_SERVICE);
            String cameraId = selectCamera(manager, cfg.lens);
            CameraCharacteristics traits = manager.getCameraCharacteristics(cameraId);
            Size size = chooseSize(traits, cfg.width, cfg.height);
            String codecName = selectHardwareCodec(size, cfg.fps);

            MediaFormat format = MediaFormat.createVideoFormat(
                MediaFormat.MIMETYPE_VIDEO_AVC, size.getWidth(), size.getHeight());
            format.setInteger(MediaFormat.KEY_COLOR_FORMAT,
                MediaCodecInfo.CodecCapabilities.COLOR_FormatSurface);
            format.setInteger(MediaFormat.KEY_BIT_RATE, cfg.bitrate);
            format.setInteger(MediaFormat.KEY_FRAME_RATE, cfg.fps);
            format.setInteger(MediaFormat.KEY_I_FRAME_INTERVAL, 2);

            encoder = MediaCodec.createByCodecName(codecName);
            encoder.configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE);
            inputSurface = encoder.createInputSurface();
            encoder.start();

            device = openCamera(manager, cameraId, cameraHandler);
            session = openSession(device, inputSurface, cameraHandler);

            CaptureRequest.Builder request = device.createCaptureRequest(CameraDevice.TEMPLATE_RECORD);
            request.addTarget(inputSurface);
            request.set(CaptureRequest.CONTROL_MODE, CaptureRequest.CONTROL_MODE_AUTO);
            Range<Integer> targetRange = selectFpsRange(traits, cfg.fps);
            if (targetRange != null) request.set(CaptureRequest.CONTROL_AE_TARGET_FPS_RANGE, targetRange);
            request.set(CaptureRequest.CONTROL_VIDEO_STABILIZATION_MODE,
                CaptureRequest.CONTROL_VIDEO_STABILIZATION_MODE_OFF);
            session.setRepeatingRequest(request.build(), null, cameraHandler);
            Log.i(TAG, "Camera started: " + cameraId + " " + size.getWidth() + "x" +
                size.getHeight() + "@" + cfg.fps + " H264 " + codecName);

            MediaCodec.BufferInfo info = new MediaCodec.BufferInfo();
            byte[] parameterSets = new byte[0];
            int frames = 0;
            while (running && !Thread.currentThread().isInterrupted() && currentClient != null &&
                   !currentClient.isClosed()) {
                int index = encoder.dequeueOutputBuffer(info, 150_000);
                if (index == MediaCodec.INFO_OUTPUT_FORMAT_CHANGED) {
                    MediaFormat output = encoder.getOutputFormat();
                    parameterSets = H264AnnexB.config(readCsd(output, "csd-0"), readCsd(output, "csd-1"));
                    Log.i(TAG, "Encoder format negotiated; SPS/PPS bytes=" + parameterSets.length);
                } else if (index >= 0) {
                    ByteBuffer buffer = encoder.getOutputBuffer(index);
                    if (buffer != null && info.size > 0) {
                        buffer.position(info.offset);
                        buffer.limit(info.offset + info.size);
                        byte[] data = new byte[info.size];
                        buffer.get(data);
                        byte[] nal = H264AnnexB.normalize(data);
                        if ((info.flags & MediaCodec.BUFFER_FLAG_CODEC_CONFIG) != 0) {
                            // Some encoders emit SPS/PPS in-band instead of csd-0/csd-1.
                            if (parameterSets.length == 0) parameterSets = nal;
                        } else {
                            if ((info.flags & MediaCodec.BUFFER_FLAG_KEY_FRAME) != 0 &&
                                    parameterSets.length > 0) {
                                destination.write(parameterSets);
                            }
                            destination.write(nal);
                            destination.flush();
                            frames++;
                            if (frames == 1 || frames % 300 == 0)
                                Log.i(TAG, "H264 frames sent: " + frames);
                        }
                    }
                    encoder.releaseOutputBuffer(index, false);
                    if ((info.flags & MediaCodec.BUFFER_FLAG_END_OF_STREAM) != 0) break;
                }
            }
        } finally {
            if (session != null) {
                try { session.stopRepeating(); } catch (Exception ignored) {}
                try { session.abortCaptures(); } catch (Exception ignored) {}
                session.close();
            }
            if (device != null) device.close();
            if (encoder != null) {
                try { encoder.stop(); } catch (Exception ignored) {}
                encoder.release();
            }
            if (inputSurface != null) inputSurface.release();
            cameraThread.quitSafely();
            cameraThread.join(1500);
        }
    }

    private static byte[] readCsd(MediaFormat format, String key) {
        try {
            ByteBuffer data = format.getByteBuffer(key);
            if (data == null) return new byte[0];
            ByteBuffer copy = data.duplicate();
            byte[] result = new byte[copy.remaining()];
            copy.get(result);
            return result;
        } catch (Exception e) {
            return new byte[0];
        }
    }

    private static String selectCamera(CameraManager manager, String lens)
            throws CameraAccessException {
        int facing = "front".equals(lens)
            ? CameraCharacteristics.LENS_FACING_FRONT : CameraCharacteristics.LENS_FACING_BACK;
        String[] ids = manager.getCameraIdList();
        if (ids.length == 0) throw new IllegalStateException("No cameras visible to Camera2");
        for (String id : ids) {
            Integer direction = manager.getCameraCharacteristics(id).get(CameraCharacteristics.LENS_FACING);
            if (direction != null && direction == facing) return id;
        }
        Log.w(TAG, "Requested lens not found, falling back to camera " + ids[0]);
        return ids[0];
    }

    private static Size chooseSize(CameraCharacteristics characteristics, int w, int h) {
        StreamConfigurationMap map = characteristics.get(
            CameraCharacteristics.SCALER_STREAM_CONFIGURATION_MAP);
        if (map == null) return new Size(w, h);
        Size[] sizes = map.getOutputSizes(MediaCodec.class);
        if (sizes == null || sizes.length == 0)
            sizes = map.getOutputSizes(MediaRecorder.class);
        if (sizes == null || sizes.length == 0) return new Size(w, h);
        Size best = sizes[0];
        long bestScore = Long.MAX_VALUE;
        for (Size size : sizes) {
            if (size.getWidth() > 3840 || size.getHeight() > 2160) continue;
            long pixelsDiff = Math.abs((long) size.getWidth() * size.getHeight() - (long) w * h);
            long ratioDiff = Math.abs((long) size.getWidth() * h - (long) size.getHeight() * w);
            long score = pixelsDiff + ratioDiff * 4;
            if (score < bestScore) {
                bestScore = score;
                best = size;
            }
        }
        return best;
    }

    private static String selectHardwareCodec(Size size, int fps) {
        for (MediaCodecInfo codec : new MediaCodecList(MediaCodecList.REGULAR_CODECS).getCodecInfos()) {
            if (!codec.isEncoder() || !codec.isHardwareAccelerated()) continue;
            try {
                MediaCodecInfo.CodecCapabilities capabilities =
                    codec.getCapabilitiesForType(MediaFormat.MIMETYPE_VIDEO_AVC);
                boolean surface = false;
                for (int color : capabilities.colorFormats) {
                    if (color == MediaCodecInfo.CodecCapabilities.COLOR_FormatSurface) {
                        surface = true;
                        break;
                    }
                }
                if (surface && capabilities.getVideoCapabilities().areSizeAndRateSupported(
                        size.getWidth(), size.getHeight(), fps))
                    return codec.getName();
            } catch (IllegalArgumentException ignored) {
                // Codec does not support H264.
            }
        }
        throw new IllegalStateException("No hardware H264 Surface encoder for " +
            size.getWidth() + "x" + size.getHeight() + "@" + fps +
            "; refusing CPU/software encoder to avoid overheating");
    }

    private static Range<Integer> selectFpsRange(CameraCharacteristics characteristics, int fps) {
        Range<Integer>[] ranges = characteristics.get(
            CameraCharacteristics.CONTROL_AE_AVAILABLE_TARGET_FPS_RANGES);
        if (ranges == null || ranges.length == 0) return null;
        Range<Integer> best = ranges[0];
        int scoreBest = Integer.MAX_VALUE;
        for (Range<Integer> r : ranges) {
            int score = Math.abs(r.getUpper() - fps) * 5 + Math.abs(r.getLower() - fps);
            if (score < scoreBest) {
                best = r;
                scoreBest = score;
            }
        }
        return best;
    }

    private static CameraDevice openCamera(CameraManager manager, String id, Handler handler)
            throws Exception {
        CountDownLatch done = new CountDownLatch(1);
        AtomicReference<CameraDevice> opened = new AtomicReference<>();
        AtomicReference<String> failure = new AtomicReference<>("Camera open timeout");
        manager.openCamera(id, new CameraDevice.StateCallback() {
            @Override public void onOpened(CameraDevice camera) {
                opened.set(camera);
                done.countDown();
            }
            @Override public void onDisconnected(CameraDevice camera) {
                failure.set("Camera disconnected");
                camera.close();
                done.countDown();
            }
            @Override public void onError(CameraDevice camera, int error) {
                failure.set("Camera error: " + error);
                camera.close();
                done.countDown();
            }
        }, handler);
        if (!done.await(10, TimeUnit.SECONDS) || opened.get() == null)
            throw new IOException(failure.get());
        return opened.get();
    }

    private static CameraCaptureSession openSession(
            CameraDevice camera, Surface surface, Handler handler) throws Exception {
        CountDownLatch done = new CountDownLatch(1);
        AtomicReference<CameraCaptureSession> configured = new AtomicReference<>();
        camera.createCaptureSession(Arrays.asList(surface), new CameraCaptureSession.StateCallback() {
            @Override public void onConfigured(CameraCaptureSession session) {
                configured.set(session);
                done.countDown();
            }
            @Override public void onConfigureFailed(CameraCaptureSession session) {
                session.close();
                done.countDown();
            }
        }, handler);
        if (!done.await(10, TimeUnit.SECONDS) || configured.get() == null)
            throw new IOException("Camera2 could not configure MediaCodec surface");
        return configured.get();
    }

    private static void closeQuietly(Closeable c) {
        try { if (c != null) c.close(); } catch (IOException ignored) {}
    }

    @Override public void onDestroy() {
        running = false;
        closeQuietly(currentClient);
        closeQuietly(listener);
        if (worker != null) {
            worker.interrupt();
            try { worker.join(1800); } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
        super.onDestroy();
    }
}
