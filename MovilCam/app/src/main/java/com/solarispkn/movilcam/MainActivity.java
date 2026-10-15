package com.solarispkn.movilcam;

import android.Manifest;
import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.graphics.Color;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.TextView;

/**
 * Control screen only. No camera preview is drawn.
 * Can be launched through ADB on a device without its physical display.
 */
public final class MainActivity extends Activity {
    private static final int CAMERA_PERMISSION_REQUEST = 10;
    private TextView status;
    private String pendingLens = "back";

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        LinearLayout column = new LinearLayout(this);
        column.setOrientation(LinearLayout.VERTICAL);
        column.setPadding(32, 32, 32, 32);
        column.setBackgroundColor(Color.rgb(22, 26, 33));

        status = new TextView(this);
        status.setTextColor(Color.WHITE);
        status.setTextSize(17);
        status.setText("MovilCam - Camera2 + H.264 + ADB. No preview.");
        column.addView(status, new LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        Button rear = new Button(this);
        rear.setText("Start rear camera");
        rear.setOnClickListener(v -> startCamera("back"));
        column.addView(rear);

        Button front = new Button(this);
        front.setText("Start front camera");
        front.setOnClickListener(v -> startCamera("front"));
        column.addView(front);

        Button stop = new Button(this);
        stop.setText("Stop camera server");
        stop.setOnClickListener(v -> {
            stopService(new Intent(this, CameraService.class));
            status.setText("MovilCam stopped.");
        });
        column.addView(stop);

        TextView help = new TextView(this);
        help.setTextColor(Color.LTGRAY);
        help.setText("Camera is inactive until OBS connects to 127.0.0.1:27183 through adb forward.\n" +
            "No microphone, local video recording, or network exposure.\n" +
            "ADB: am start -n com.solarispkn.movilcam/.MainActivity --ez autostart true --es lens back");
        column.addView(help);
        setContentView(column);

        applyIntent(getIntent());
    }

    @Override protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        applyIntent(intent);
    }

    private void applyIntent(Intent intent) {
        if (intent != null && intent.getBooleanExtra("autostart", false)) {
            String lens = intent.getStringExtra("lens");
            // Wait until Activity is visible so Android permits the camera foreground service.
            new Handler(Looper.getMainLooper()).postDelayed(
                () -> startCamera("front".equalsIgnoreCase(lens) ? "front" : "back"), 350);
        }
    }

    private void startCamera(String lens) {
        pendingLens = lens;
        if (checkSelfPermission(Manifest.permission.CAMERA) != PackageManager.PERMISSION_GRANTED) {
            status.setText("Camera permission required. Grant in Android settings or via adb pm grant.");
            requestPermissions(new String[]{Manifest.permission.CAMERA}, CAMERA_PERMISSION_REQUEST);
            return;
        }
        Intent service = new Intent(this, CameraService.class);
        service.putExtra("lens", lens);
        Intent launch = getIntent();
        service.putExtra("width", safeInt(launch, "width", 1920, 320, 3840));
        service.putExtra("height", safeInt(launch, "height", 1080, 240, 2160));
        service.putExtra("fps", safeInt(launch, "fps", 60, 5, 60));
        service.putExtra("bitrate", safeInt(launch, "bitrate", 12_000_000, 500_000, 24_000_000));
        try {
            startForegroundService(service);
            status.setText("MovilCam ready (" + lens + "). Waiting for OBS USB client. " +
                "See adb logcat -s MovilCam:*");
        } catch (RuntimeException e) {
            status.setText("Could not start: " + e.getMessage());
        }
    }

    private static int safeInt(Intent i, String name, int fallback, int lo, int hi) {
        int v = i == null ? fallback : i.getIntExtra(name, fallback);
        return Math.max(lo, Math.min(hi, v));
    }

    @Override public void onRequestPermissionsResult(
            int requestCode, String[] permissions, int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode == CAMERA_PERMISSION_REQUEST && grantResults.length > 0 &&
                grantResults[0] == PackageManager.PERMISSION_GRANTED) {
            startCamera(pendingLens);
        }
    }
}
