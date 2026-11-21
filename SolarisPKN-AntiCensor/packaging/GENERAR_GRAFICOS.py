# -*- coding: utf-8 -*-
"""Original SVG/GIF visual illustrations for public AntiCensor docs."""
from pathlib import Path
from math import sin, pi
from PIL import Image, ImageDraw, ImageFont
import html

ROOT = Path(__file__).resolve().parents[1]
ASSETS=ROOT / "docs" / "assets"
ASSETS.mkdir(parents=True,exist_ok=True)
GREEN="#82ffd4";BLUE="#67c5fc";PURPLE="#a693ff";DARK="#10141f";SOFT="#b8c4d7"

def line(x1,y1,x2,y2,color=GREEN,width=3):
    return f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="{width}" stroke-linecap="round"/>'

def text(x,y,label,size=20,color="#ffffff",weight="400",anchor="start"):
    return f'<text x="{x}" y="{y}" fill="{color}" font-family="Segoe UI,Arial,sans-serif" font-weight="{weight}" font-size="{size}" text-anchor="{anchor}">{html.escape(label)}</text>'

def rect(x,y,w,h,fill="#192333",rx=16,stroke="#32465c"):
    return f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{rx}" fill="{fill}" stroke="{stroke}" stroke-width="1.4"/>'

def svg(w,h,children):
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" role="img" aria-label="SolarisPKN AntiCensor visual guide">
<defs>
 <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#101420"/><stop offset=".55" stop-color="#171f34"/><stop offset="1" stop-color="#231a3b"/></linearGradient>
 <linearGradient id="glow" x1="0" y1="0" x2="1" y2="0"><stop stop-color="#82ffd4"/><stop offset="1" stop-color="#67c5fc"/></linearGradient>
 <filter id="soft"><feGaussianBlur stdDeviation="36"/></filter>
</defs>
<rect x="0" y="0" width="{w}" height="{h}" rx="22" fill="url(#bg)"/>
<circle cx="{int(w*.88)}" cy="{int(h*.2)}" r="135" fill="#267d78" opacity=".19" filter="url(#soft)"/>
{children}
</svg>'''

# Hero banner: dual-language tagline, vector waves and visual mic.
w,h=1200,360
a=[]
a.append('<rect x="34" y="29" width="236" height="38" rx="19" fill="#243d42" stroke="#3a8378"/>')
a.append(text(152,54,"🎙 SOLARISPKN LABS",15,GREEN,"700","middle"))
a.append(text(44,137,"AntiCensor",72,"#ffffff","800"))
a.append(text(48,185,"Tu voz controla Lelele. Tu voz real no sale.",25,SOFT,"400"))
a.append(text(48,218,"Your voice drives Lelele. Your real voice stays private.",22,SOFT))
a.append(rect(47,265,153,42,"#213b3a",21,"#3a8174"))
a.append(text(124,293,"VOICE GATE",16,GREEN,"700","middle"))
a.append(rect(212,265,153,42,"#212f45",21,"#355a84"))
a.append(text(288,293,"WINDOWS",16,BLUE,"700","middle"))
a.append(rect(377,265,182,42,"#302847",21,"#6556a0"))
a.append(text(467,293,"V0.8.5",16,PURPLE,"700","middle"))
# Stylized mic and waveform
a.append('<circle cx="935" cy="164" r="138" fill="#1b3f42" opacity=".7"/>')
a.append('<circle cx="935" cy="164" r="119" fill="none" stroke="#3b9188" stroke-dasharray="5 12" stroke-width="2"/>')
a.append('<rect x="909" y="86" width="52" height="112" rx="25" fill="url(#glow)"/>')
a.append('<path d="M884 159 A51 51 0 0 0 986 159" fill="none" stroke="#f4faff" stroke-width="10" stroke-linecap="round"/>')
a.append(line(935,211,935,239,"#f4faff",9))
a.append(line(905,242,965,242,"#f4faff",9))
for k in range(-5,6):
    if abs(k)<2:continue
    x=935+k*37
    a.append(line(x,160-12*(1+abs(sin(k))),x,160+12*(1+abs(sin(k))),GREEN if k<0 else BLUE,5))
(ASSETS/"hero.svg").write_text(svg(w,h,"\n".join(a)),encoding="utf-8")

def architecture(lang):
    es=lang=="es"
    h=450;w=1280
    a=[text(39,62,"🎮  Cómo funciona / How it works" if es else "🎮  How it works / Cómo funciona",29,"#ffffff","700")]
    tracks=[
      (42,112,"Micrófono físico" if es else "Physical microphone","Gadnic / Realtek",GREEN),
      (337,112,"Filtro de voz" if es else "Voice activity gate","WebRTC VAD 20 ms",BLUE),
      (632,112,"Síntesis Lelele" if es else "Synthetic Lelele","Generated audio",PURPLE),
      (927,112,"Micrófono virtual" if es else "Virtual microphone","VB-CABLE → Roblox",GREEN)
    ]
    for x,y,label,sub,color in tracks:
        a.append(rect(x,y,261,114))
        a.append(f'<rect x="{x+14}" y="{y+18}" width="6" height="75" rx="3" fill="{color}"/>')
        a.append(text(x+34,y+46,label,20,"#ffffff","700"))
        a.append(text(x+34,y+79,sub,17,SOFT))
    for x in (306,602,897):
        a.append(line(x,169,x+22,169,GREEN,4))
        a.append(f'<path d="M{x+18} 163 l8 6 -8 6" fill="none" stroke="{GREEN}" stroke-width="3" stroke-linejoin="round"/>')
    a.append(rect(42,296,549,106,"#252236",14,"#655384"))
    a.append(text(64,338,"🔒  NO se envía tu voz real" if es else "🔒  Your real speech is NOT forwarded",23,"#d5caff","700"))
    a.append(text(64,374,"Solo se utiliza para detectar actividad / volumen" if es else "Only speech activity / loudness is measured",16,SOFT))
    a.append(rect(645,296,543,106,"#192939",14,"#47738b"))
    a.append(text(663,338,"🎧  El sonido del juego va por separado" if es else "🎧  Game sound stays on a separate path",22,BLUE,"700"))
    a.append(text(665,374,"Sonar Gaming → auriculares / headphones",16,SOFT))
    a.append('<path d="M168 227 L168 266 L168 288" stroke="#ac89ea" stroke-width="3" stroke-dasharray="5 7" fill="none"/>')
    a.append('<path d="M815 268 L815 288" stroke="#5993be" stroke-width="3" stroke-dasharray="5 7" fill="none"/>')
    (ASSETS/f"audio-flow.{lang}.svg").write_text(svg(w,h,"\n".join(a)),encoding="utf-8")
architecture("es");architecture("en")

def features(lang):
    es=lang=="es";w,h=1200,330
    a=[text(42,53,"✨  Qué hace AntiCensor" if es else "✨  What AntiCensor does",31,"#ffffff","700")]
    caps=[
        ("🎙","Voz, no ruido" if es else "Speech, not noise",
         "Detector clásico sin transcripción" if es else "Classic detector. No transcription",GREEN),
        ("🤖","LELELE sintético" if es else "Synthetic LELELE",
         "El juego recibe un efecto, no tu voz" if es else "Games receive an effect, not your voice",PURPLE),
        ("🛠","Control experto" if es else "Expert controls",
         "Inventario VB-CABLE opcional" if es else "Optional VB-CABLE diagnostics",BLUE)
    ]
    for i,(ico,title,sub,c) in enumerate(caps):
        x=40+i*390
        a.append(rect(x,95,362,198))
        a.append(f'<circle cx="{x+62}" cy="151" r="37" fill="#28413f"/>')
        a.append(text(x+62,165,ico,33,c,"400","middle"))
        a.append(text(x+24,225,title,22,"#ffffff","700"))
        a.append(text(x+24,260,sub,15,SOFT))
    (ASSETS/f"features.{lang}.svg").write_text(svg(w,h,"\n".join(a)),encoding="utf-8")
features("es");features("en")

# GIF: full-resolution original educational demonstration, no UI screenshots.
W,H=940,420
try:
    font=ImageFont.truetype(r"C:\Windows\Fonts\segoeui.ttf",20)
    fontbig=ImageFont.truetype(r"C:\Windows\Fonts\segoeuib.ttf",32)
    fontsmall=ImageFont.truetype(r"C:\Windows\Fonts\segoeui.ttf",16)
except OSError:
    font=fontbig=fontsmall=ImageFont.load_default()
frames=[]
for idx in range(24):
    im=Image.new("RGB",(W,H),(15,20,32))
    d=ImageDraw.Draw(im)
    d.rounded_rectangle((20,18,W-20,H-20),radius=24,fill=(24,31,48),outline=(58,74,94),width=2)
    voice=idx>=10 and idx<19
    stage="2 / VOICE • VOZ" if voice else ("1 / NOISE • RUIDO" if idx<10 else "3 / SILENCE • SILENCIO")
    d.text((48,45),"🎙" if False else "SolarisPKN AntiCensor  |  LELELE",font=fontbig,fill=(224,255,245))
    d.text((49,91),"DEMO / DEMOSTRACIÓN — ilustración, no captura real",font=fontsmall,fill=(165,183,204))
    d.rounded_rectangle((48,128,890,173),radius=12,fill=(31,42,60))
    d.text((67,137),stage,font=font,fill=(140,255,215) if voice else (197,188,239))
    labels=[("INPUT / ENTRADA",220),("VOICE GATE / FILTRO",291),("LELELE / SALIDA",362)]
    for k,(label,yy) in enumerate(labels):
        d.text((48,yy-17),label,font=fontsmall,fill=(180,200,217))
        left=295;right=860
        d.rounded_rectangle((left,yy-9,right,yy+10),radius=8,fill=(35,47,65))
        for j in range(32):
            pulse=abs(sin((idx*0.7)+(j*0.95))) * (0.35+0.6*abs(sin(j*0.43+idx*0.20)))
            if k==0:
                height=12+int(pulse*(voice and 37 or 22))
                col=(124,179,248) if voice else (189,168,244)
            else:
                height=12+int(pulse*(33 if voice else 0))
                col=(112,250,196) if voice else (66,80,102)
            cx=left+8+j*17
            d.rounded_rectangle((cx,yy-min(23,height//2),cx+7,yy+min(23,height//2)),radius=3,fill=col)
    d.text((48,390),"Noise stays closed  •  Speech opens gate  •  No words are transcribed",font=fontsmall,fill=(173,190,205))
    frames.append(im.quantize(colors=80,method=Image.Quantize.FASTOCTREE))
out=ASSETS/"voice-gate-demo.gif"
# Compact loop for GitHub README and offline Windows installer, preserving motion.
web_frames = [
    f.convert("RGB").resize((400, 179), Image.Resampling.LANCZOS).quantize(
        colors=32, method=Image.Quantize.FASTOCTREE
    ) for f in frames[::4]
]
web_frames[0].save(out,save_all=True,append_images=web_frames[1:],
                   duration=720,loop=0,optimize=True,disposal=2)
print("ASSETS")
for f in sorted(ASSETS.iterdir()):
    print(f.name,f.stat().st_size)
