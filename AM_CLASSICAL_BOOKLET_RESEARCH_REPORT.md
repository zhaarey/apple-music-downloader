# Apple Music Classical: Digital Booklet Reverse Engineering & Extraction Report

This report documents the end-to-end technical findings for discovering, retrieving, and decrypting Apple Music Classical digital album booklets (PDFs).

---

## 1. Overview & Architecture

Unlike standard Apple Music (which does not provide in-app digital booklet viewers for Classical releases), **Apple Music Classical** (iOS and Android) features a dedicated digital album booklet viewer.

Digital booklets are:
1. **Discoverable** via the Classical Context Menu API.
2. **Protected at the server layer** via Google Play Integrity attestation (on Android) or App Attest (on iOS).
3. **Encrypted in transit** via chunked AES-CBC with MD5-derived initialization vectors (IVs).
4. **Decrypted and cached locally** on the device under the app's internal storage cache.

---

## 2. API Endpoints

### Step 1: Booklet Discovery (Context Menu)

To check if an album has a digital booklet:

```http
GET https://classical.music.apple.com/api/classical/v10/query/context-menu/{storefront}?entityId={albumId}&entityType=album
Authorization: Bearer {developer_token}
```

*(Note: Also supported on `https://classical-api.music.apple.com/v1/query/context-menu/{storefront}?entityId={albumId}&entityType=album`)*

#### Response Snippet:
```json
{
  "data": {
    "component": {
      "items": [
        {
          "subtype": "booklet",
          "icon": "book",
          "title": "View Album Booklet",
          "action": {
            "title": "Ravel: Piano Concertos",
            "type": "booklet",
            "url": "/query/booklet/in/1823823405",
            "message": "Booklet provided by Alpha Classics"
          }
        }
      ]
    }
  }
}
```

- If an item with `"subtype": "booklet"` exists, the album provides a digital booklet.
- The `action.url` path (e.g. `/query/booklet/in/1823823405`) is used for fetching and cache indexing.

---

### Step 2: Booklet Keys & Download URL Endpoint

```http
POST https://classical-api.music.apple.com/v1/query/booklet/{storefront}/{albumId}
Authorization: Bearer {developer_token}
Content-Type: application/json

{
  "payload": "<Google_Play_Integrity_Token>"
}
```

#### Attestation Requirement:
- **Header/Body**: The body requires `AlbumBookletDetailsRequestDto(payload = "...")`.
- On Android, the app requests this token via **Google Play Integrity API** (`requestExpressIntegrityToken`).
- **Validation**: Apple validates that the token comes from Google verifying:
  - Official package: `com.apple.android.music.classical`
  - Signed by Apple's release key
  - Device meets integrity (`MEETS_DEVICE_INTEGRITY` / `MEETS_BASIC_INTEGRITY`).
- **If unauthenticated / invalid**: The server responds with `HTTP 400`:
  ```json
  {"type": "screen-error", "title": "Something went wrong."}
  ```

#### Successful Response:
```json
{
  "url": "https://.../booklet.enc",
  "encryptionKey": "<base64_or_hex_aes_key>",
  "encryptionSalt": "<base64_or_hex_salt>"
}
```

---

## 3. Decryption Algorithm (Reverse-Engineered from APK)

The decryption routine in the Android client (`LF5/l` in bytecode) processes the encrypted payload as follows:

- **Cipher**: `AES/CBC/NoPadding`
- **Key**: Derived from `encryptionKey`
- **Chunk Size**: 2048 bytes (or sequential stream chunks)
- **IV Derivation**:
  For each chunk index $i$ ($i = 0, 1, 2, \dots$):
  1. Allocate a 20-byte buffer.
  2. Copy the 16-byte `encryptionSalt` into bytes `0..15`.
  3. Write chunk index $i$ as a 32-bit big-endian integer into bytes `16..19`.
  4. Compute `MD5(buffer)` $\rightarrow$ yields a 16-byte hash.
  5. Use this 16-byte MD5 digest as the AES-CBC IV for chunk $i$.

---

## 4. Android Client Cache Architecture

When the user taps "View Album Booklet" in the Classical app, the app decrypts the booklet and caches it in internal storage:

```
/data/user/0/com.apple.android.music.classical/cache/booklets/{hash}/decrypted-{hash}.pdf
```

### Cache Directory Hash Formula
The `{hash}` folder name is calculated using Java's standard `java.lang.String.hashCode()` on the `action.url` path:

```go
func JavaHashCode(s string) int32 {
    var h int32
    for i := 0; i < len(s); i++ {
        h = 31*h + int32(s[i])
    }
    return h
}
```

#### Verified Example:
- **Booklet Path**: `"/query/booklet/in/1823823405"`
- `JavaHashCode("/query/booklet/in/1823823405")` $\rightarrow$ `729783711`
- **Target File on Phone**:
  `/data/user/0/com.apple.android.music.classical/cache/booklets/729783711/decrypted-729783711.pdf`
- **Extracted Content**: 25-page high-resolution booklet for *Ravel: Piano Concertos* (Nelson Goerner / Orchestre Philharmonique De Monte-Carlo / Kazuki Yamada / Alpha Classics).

---

## 5. Why a Headless PC Wrapper Cannot Download Booklets Directly

Many users wondered why `wrapper-lite` (or a similar headless QEMU wrapper) can decrypt lossless ALAC and Dolby Atmos audio without a phone, but **cannot** download Classical booklets:

| Feature | Audio Decryption (ALAC / Atmos) | Classical Booklets (PDF) |
|---|---|---|
| **Underlying Tech** | Apple FairPlay DRM (`libCoreFP.so`) | Google Play Integrity Attestation |
| **API Provider** | Apple Music Android native C++ libs | Google Play Services (GMS) |
| **Hardware Binding** | Emulated in bionic/QEMU userland | Cryptographic hardware Keystore |
| **Headless PC Feasible?** | **Yes** (wrapper-lite works on PC) | **No** (requires valid Play Integrity) |

On rooted Android devices, passing Play Integrity requires:
- **Play Integrity Fix (PIF)** to spoof build properties/fingerprints.
- **TrickyStore** to intercept Keystore attestation and sign requests using a genuine OEM Keybox certificate.
- **Google Play Services (GMS)** running inside the Android framework.

A headless C++ QEMU container running bare Linux/Android shared libraries lacks ART, GMS, Keystore, and TrickyStore, making independent token generation impossible on PC.

---

## 6. End-to-End Extraction Workflow (via ADB)

If an Android device (rooted with Magisk/KernelSU + Play Integrity Fix / TrickyStore) is connected via ADB, the booklet can be automatically extracted:

```bash
# 1. Check if the album booklet is already cached on the device
adb shell su -c "ls /data/user/0/com.apple.android.music.classical/cache/booklets/729783711/decrypted-729783711.pdf"

# 2. If not cached, launch the Classical album deep-link
adb shell am start -a android.intent.action.VIEW -d "https://classical.music.apple.com/in/album/1823823405"

# 3. Tap the "View Album Booklet" icon (top-right bar: resource-id "topBarBookletButton")
adb shell input tap 780 185

# 4. Wait ~2-3 seconds for download & decryption, then pull the clean PDF
adb exec-out su -c "cat /data/user/0/com.apple.android.music.classical/cache/booklets/729783711/decrypted-729783711.pdf" > Booklet.pdf

# 5. Dismiss the booklet viewer on the device
adb shell input keyevent KEYCODE_BACK
```

---

## 7. Summary for Downloader Developers

1. **Detection**: Query `/api/classical/v10/query/context-menu/{storefront}?entityId={albumId}&entityType=album` to know if a booklet exists.
2. **Integrity Barrier**: Booklets cannot currently be downloaded via pure CLI/curl or `wrapper-lite` because Apple enforces Google Play Integrity on the booklet API endpoint.
3. **Automated Helper**: A connected Android phone with root can serve as an automated companion to fetch and decrypt the booklet via ADB, which the downloader can pull directly into the album output folder.
