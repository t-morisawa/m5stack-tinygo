# M5Stack Core2 プロジェクト

M5Stack Core2 で加速度センサー(BMI270)を使い、Gopherの画像が傾きに応じて動くデモアプリ。

## プロジェクト構成

```
sketch_jun7a/     Arduino IDE版（C++, M5Unified使用）
tinygo/           TinyGo版
├── gravity/      加速度で画像が動くデモアプリ（メイン）
│   ├── bmi270/   BMI270ドライバ
│   └── gravity.go
├── display.go    ディスプレイ描画テスト
└── main.go       Hello World（動作確認用）
```

## 環境

- ボード: M5Stack Core2 (v1.1 / BMI270搭載)
- TinyGo: target `m5stack-core2`
- シリアルポート: `/dev/cu.usbserial-*` (環境に応じて変更)

## ビルド & 書き込み (TinyGo)

### gravity (加速度デモ)

```bash
# ビルドのみ
tinygo build -target=m5stack-core2 -o gravity.bin ./gravity

# ビルド & 書き込み
tinygo flash -target=m5stack-core2 -port=/dev/cu.usbserial-5B1F0081741 ./gravity
```

### display (描画テスト)

```bash
tinygo flash -target=m5stack-core2 -port=/dev/cu.usbserial-5B1F0081741 .
```

### シリアルモニタ

```bash
tinygo monitor -port=/dev/cu.usbserial-5B1F0081741
```

## ビルド & 書き込み (Arduino IDE)

`sketch_jun7a/sketch_jun7a.ino` を Arduino IDE で開き、ボード「M5Stack-Core2」を選択して書き込み。

必要なライブラリ:
- M5Unified
- M5GFX
