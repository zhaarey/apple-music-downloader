export namespace engine {
	
	export class DependencyStatus {
	    name: string;
	    ok: boolean;
	    detail: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DependencyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	        this.required = source["required"];
	    }
	}

}

export namespace structs {
	
	export class ConfigSet {
	    storefront: string;
	    "media-user-token": string;
	    "authorization-token": string;
	    language: string;
	    "save-lrc-file": boolean;
	    "lrc-type": string;
	    "lrc-format": string;
	    "save-animated-artwork": boolean;
	    "emby-animated-artwork": boolean;
	    "embed-lrc": boolean;
	    "embed-cover": boolean;
	    "save-artist-cover": boolean;
	    "cover-size": string;
	    "cover-format": string;
	    "tag-sort-order": boolean;
	    "tag-itunes-id": boolean;
	    "alac-save-folder": string;
	    "atmos-save-folder": string;
	    "aac-save-folder": string;
	    "mv-save-folder": string;
	    "album-folder-format": string;
	    "playlist-folder-format": string;
	    "artist-folder-format": string;
	    "song-file-format": string;
	    "explicit-choice": string;
	    "clean-choice": string;
	    "apple-master-choice": string;
	    "max-memory-limit": number;
	    "decrypt-m3u8-port": string;
	    "get-m3u8-port": string;
	    "get-m3u8-mode": string;
	    "get-m3u8-from-device": boolean;
	    "aac-type": string;
	    "alac-max": number;
	    "atmos-max": number;
	    "limit-max": number;
	    "use-songinfo-for-playlist": boolean;
	    "dl-albumcover-for-playlist": boolean;
	    "mv-audio-type": string;
	    "mv-max": number;
	    "convert-after-download": boolean;
	    "convert-format": string;
	    "convert-keep-original": boolean;
	    "convert-skip-if-source-matches": boolean;
	    "ffmpeg-path": string;
	    "convert-extra-args": string;
	    "convert-with-metadata": boolean;
	    "convert-warn-lossy-to-lossless": boolean;
	    "convert-skip-lossy-to-lossless": boolean;
	    "convert-check-bad-alac": boolean;
	    "convert-delete-bad-alac": boolean;
	    "alac-fix": boolean;
	    "exit-on-error": boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storefront = source["storefront"];
	        this["media-user-token"] = source["media-user-token"];
	        this["authorization-token"] = source["authorization-token"];
	        this.language = source["language"];
	        this["save-lrc-file"] = source["save-lrc-file"];
	        this["lrc-type"] = source["lrc-type"];
	        this["lrc-format"] = source["lrc-format"];
	        this["save-animated-artwork"] = source["save-animated-artwork"];
	        this["emby-animated-artwork"] = source["emby-animated-artwork"];
	        this["embed-lrc"] = source["embed-lrc"];
	        this["embed-cover"] = source["embed-cover"];
	        this["save-artist-cover"] = source["save-artist-cover"];
	        this["cover-size"] = source["cover-size"];
	        this["cover-format"] = source["cover-format"];
	        this["tag-sort-order"] = source["tag-sort-order"];
	        this["tag-itunes-id"] = source["tag-itunes-id"];
	        this["alac-save-folder"] = source["alac-save-folder"];
	        this["atmos-save-folder"] = source["atmos-save-folder"];
	        this["aac-save-folder"] = source["aac-save-folder"];
	        this["mv-save-folder"] = source["mv-save-folder"];
	        this["album-folder-format"] = source["album-folder-format"];
	        this["playlist-folder-format"] = source["playlist-folder-format"];
	        this["artist-folder-format"] = source["artist-folder-format"];
	        this["song-file-format"] = source["song-file-format"];
	        this["explicit-choice"] = source["explicit-choice"];
	        this["clean-choice"] = source["clean-choice"];
	        this["apple-master-choice"] = source["apple-master-choice"];
	        this["max-memory-limit"] = source["max-memory-limit"];
	        this["decrypt-m3u8-port"] = source["decrypt-m3u8-port"];
	        this["get-m3u8-port"] = source["get-m3u8-port"];
	        this["get-m3u8-mode"] = source["get-m3u8-mode"];
	        this["get-m3u8-from-device"] = source["get-m3u8-from-device"];
	        this["aac-type"] = source["aac-type"];
	        this["alac-max"] = source["alac-max"];
	        this["atmos-max"] = source["atmos-max"];
	        this["limit-max"] = source["limit-max"];
	        this["use-songinfo-for-playlist"] = source["use-songinfo-for-playlist"];
	        this["dl-albumcover-for-playlist"] = source["dl-albumcover-for-playlist"];
	        this["mv-audio-type"] = source["mv-audio-type"];
	        this["mv-max"] = source["mv-max"];
	        this["convert-after-download"] = source["convert-after-download"];
	        this["convert-format"] = source["convert-format"];
	        this["convert-keep-original"] = source["convert-keep-original"];
	        this["convert-skip-if-source-matches"] = source["convert-skip-if-source-matches"];
	        this["ffmpeg-path"] = source["ffmpeg-path"];
	        this["convert-extra-args"] = source["convert-extra-args"];
	        this["convert-with-metadata"] = source["convert-with-metadata"];
	        this["convert-warn-lossy-to-lossless"] = source["convert-warn-lossy-to-lossless"];
	        this["convert-skip-lossy-to-lossless"] = source["convert-skip-lossy-to-lossless"];
	        this["convert-check-bad-alac"] = source["convert-check-bad-alac"];
	        this["convert-delete-bad-alac"] = source["convert-delete-bad-alac"];
	        this["alac-fix"] = source["alac-fix"];
	        this["exit-on-error"] = source["exit-on-error"];
	    }
	}

}

