'use strict';
setTimeout(() => {
    global.getM3U8fromDownload = function(adamID) {
        var C3282k = Java.use("c.a.a.e.o.k");
        var m7125s = C3282k.a().s();
        var PurchaseRequest$PurchaseRequestPtr = Java.use("com.apple.android.storeservices.javanative.account.PurchaseRequest$PurchaseRequestPtr");

        var c3249t = Java.cast(m7125s, Java.use("c.a.a.e.k.t"));
        var create = PurchaseRequest$PurchaseRequestPtr.create(c3249t.n.value);
        create.get().setProcessDialogActions(true);
        create.get().setURLBagKey("subDownload");
        create.get().setBuyParameters(`salableAdamId=${adamID}&price=0&pricingParameters=SUBS&productType=S`);
        create.get().run();
        var response = create.get().getResponse();
        if (response.get().getError().get() == null) {
            var item = response.get().getItems().get(0);
            var assets = item.get().getAssets();
            var size = assets.size();
            return assets.get(size - 1).get().getURL();
        } else {
            return response.get().getError().get().errorCode();
        }
    };
    global.getM3U8 = function(adamID) {
        Java.use("com.apple.android.music.common.MainContentActivity");
        var SVPlaybackLeaseManagerProxy;
        Java.choose("com.apple.android.music.playback.SVPlaybackLeaseManagerProxy", {
            onMatch: function (x) {
                SVPlaybackLeaseManagerProxy = x
            },
            onComplete: function (x) {}
        });
        var HLSParam = Java.array('java.lang.String', ["HLS"])
        try {
            var MediaAssetInfo = SVPlaybackLeaseManagerProxy.requestAsset(parseInt(adamID), HLSParam, false);
            if (MediaAssetInfo === null) {
                return -1;
            }
            return MediaAssetInfo.getDownloadUrl();
        } catch (e) {
            console.log("Error calling requestAsset:", e);
            return -1;
        }
    };
    
    function performJavaOperations(adamID) {
        return new Promise((resolve, reject) => {
            Java.performNow(function () {
                const url = getM3U8fromDownload(adamID);
                resolve(url);
            });
        });
    }
    
    async function handleM3U8Connection(s) {
        console.log("New M3U8 connection!");
        try {
            const adamSize = (await s.input.readAll(1)).unwrap().readU8();
            if (adamSize !== 0) {
                const adam = await s.input.readAll(adamSize);
                const byteArray = new Uint8Array(adam);
                let adamID = "";
                for (let i = 0; i < byteArray.length; i++) {
                    adamID += String.fromCharCode(byteArray[i]);
                }
                console.log("adamID:", adamID);
                let m3u8Url;
                try {
                    m3u8Url = await performJavaOperations(adamID);
                    console.log("M3U8 URL: ", m3u8Url);
                    const m3u8Array = stringToByteArray(m3u8Url + "\n");
                    await s.output.writeAll(m3u8Array);
                } catch (error) {
                    console.error("Error performing Java operations:", error);
                }
            }
        } catch (err) {
            console.error("Error handling M3U8 connection:", err);
        } finally {
            await s.close();
        }
    }

    
    const stringToByteArray = str => {
        const byteArray = [];
        for (let i = 0; i < str.length; ++i) {
            byteArray.push(str.charCodeAt(i));
        }
        return byteArray;
    };
    
    Socket.listen({
        family: "ipv4",
        host: "0.0.0.0",
        port: 20020,
    }).then(async function (listener) {
        while (true) {
            const connection = await listener.accept();
            handleM3U8Connection(connection).catch(console.error); 
        }
    }).catch(console.log);
}, 4000);
