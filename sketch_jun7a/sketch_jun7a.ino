#include <M5Unified.h>

extern const unsigned char gopher[2043];

static float ax = 0, ay = 0, az = 0;
static LGFX_Sprite* sprite = nullptr;
static LGFX_Sprite* gopherSprite = nullptr;
static LGFX_Sprite* textSprite = nullptr;

static constexpr int img_w = 60;
static constexpr int img_h = 60;
static constexpr int max_offset = 80;
static constexpr int text_w = 240;
static constexpr int text_h = 30;

void setup(void) {
    auto cfg = M5.config();
    M5.begin(cfg);

    sprite = new LGFX_Sprite(&M5.Display);
    sprite->setPsram(true);
    sprite->setColorDepth(8);
    if (!sprite->createSprite(M5.Display.width(), M5.Display.height())) {
        M5.Display.printf("Sprite create failed!");
        delete sprite;
        sprite = nullptr;
        return;
    }

    gopherSprite = new LGFX_Sprite(&M5.Display);
    gopherSprite->setPsram(true);
    gopherSprite->setColorDepth(16);
    if (!gopherSprite->createSprite(img_w, img_h)) {
        M5.Display.printf("Gopher sprite create failed!");
        delete gopherSprite;
        gopherSprite = nullptr;
        return;
    }
    gopherSprite->drawJpg(gopher, sizeof(gopher), 0, 0, img_w, img_h);
    gopherSprite->setPivot(img_w / 2.0f, img_h / 2.0f);

    textSprite = new LGFX_Sprite(&M5.Display);
    textSprite->setPsram(true);
    textSprite->setColorDepth(16);
    if (!textSprite->createSprite(text_w, text_h)) {
        M5.Display.printf("Text sprite create failed!");
        delete textSprite;
        textSprite = nullptr;
        return;
    }
    textSprite->fillSprite(TFT_WHITE);
    textSprite->setTextFont(&fonts::Font4);
    textSprite->setTextSize(1);
    textSprite->setTextColor(TFT_BLACK);
    textSprite->setCursor(0, 0);
    textSprite->print("Hello, TinyGo World");
    textSprite->setPivot(text_w / 2.0f, text_h / 2.0f);
}

void loop(void) {
    if (M5.Imu.update()) {
        auto data = M5.Imu.getImuData();
        ax = data.accel.x;
        ay = data.accel.y;
        az = data.accel.z;
    }

    if (sprite == nullptr || gopherSprite == nullptr || textSprite == nullptr) return;

    int dw = M5.Display.width();
    int dh = M5.Display.height();

    float clamp_x = constrain(ax, -2.0f, 2.0f) / 2.0f;
    float clamp_y = constrain(ay, -2.0f, 2.0f) / 2.0f;

    int gx = dw / 2 + (int)(clamp_x * max_offset);
    int gy = dh / 2 + (int)(clamp_y * max_offset);

    float angle_deg = atan2(ax, ay) * 180.0f / PI;

    float len = sqrtf(ax * ax + ay * ay);
    float offset_dist = 60.0f;
    float ux = 0, uy = -1;
    if (len > 0.01f) {
        ux = -ax / len;
        uy = -ay / len;
    }
    int tx = gx + (int)(ux * offset_dist);
    int ty = gy + (int)(uy * offset_dist);

    sprite->fillSprite(TFT_WHITE);
    gopherSprite->pushRotateZoom(sprite, gx, gy, angle_deg, 1.0f, 1.0f);
    textSprite->pushRotateZoom(sprite, tx, ty, angle_deg, 1.0f, 1.0f);
    sprite->pushSprite(0, 0);
    delay(16);
}