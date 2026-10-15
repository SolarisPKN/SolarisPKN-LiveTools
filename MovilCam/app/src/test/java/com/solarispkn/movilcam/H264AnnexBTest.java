package com.solarispkn.movilcam;
import org.junit.Test;
import static org.junit.Assert.*;
public class H264AnnexBTest {
  @Test public void rawNaluGetsPrefix() {
    assertArrayEquals(new byte[]{0,0,0,1,0x67,0x42},
        H264AnnexB.normalize(new byte[]{0x67,0x42}));
  }
  @Test public void alreadyAnnexBRemainsUnchanged() {
    byte[] data = {0,0,0,1,0x68,1,2};
    assertSame(data,H264AnnexB.normalize(data));
  }
  @Test public void avccLengthsBecomeStartCodes() {
    byte[] data={0,0,0,2,0x67,0x42,0,0,0,1,0x68};
    assertArrayEquals(new byte[]{0,0,0,1,0x67,0x42,0,0,0,1,0x68},
        H264AnnexB.normalize(data));
  }
  @Test public void parameterSetsAreCombined() {
    assertArrayEquals(new byte[]{0,0,0,1,0x67,1,0,0,0,1,0x68,2},
        H264AnnexB.config(new byte[]{0x67,1},new byte[]{0x68,2}));
  }
  @Test public void emptyDoesNotCrash() {
    assertEquals(0,H264AnnexB.normalize(new byte[0]).length);
  }
}
