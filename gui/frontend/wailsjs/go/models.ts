export namespace apple {
	
	export class PreviewTrack {
	    num: number;
	    id: string;
	    name: string;
	    artist: string;
	    type: string;
	    duration: string;
	    duration_ms: number;
	    explicit: boolean;
	    is_mv: boolean;
	    url?: string;
	    art_url?: string;
	    album?: string;
	    album_artist?: string;
	    genre?: string;
	    year?: string;
	    track_number?: number;
	    disc_number?: number;
	
	    static createFrom(source: any = {}) {
	        return new PreviewTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.artist = source["artist"];
	        this.type = source["type"];
	        this.duration = source["duration"];
	        this.duration_ms = source["duration_ms"];
	        this.explicit = source["explicit"];
	        this.is_mv = source["is_mv"];
	        this.url = source["url"];
	        this.art_url = source["art_url"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.disc_number = source["disc_number"];
	    }
	}
	export class PreviewResult {
	    url: string;
	    type: string;
	    title: string;
	    subtitle: string;
	    art_url: string;
	    track_count: number;
	    total_duration: string;
	    tracks: PreviewTrack[];
	    can_select_tracks: boolean;
	    output_folder: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.art_url = source["art_url"];
	        this.track_count = source["track_count"];
	        this.total_duration = source["total_duration"];
	        this.tracks = this.convertValues(source["tracks"], PreviewTrack);
	        this.can_select_tracks = source["can_select_tracks"];
	        this.output_folder = source["output_folder"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

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

export namespace media {
	
	export class AudioTagInfo {
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    track_total: number;
	    disc_number: number;
	    disc_total: number;
	    has_artwork: boolean;
	    artwork_mime?: string;
	    artwork_b64?: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new AudioTagInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.track_total = source["track_total"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	        this.has_artwork = source["has_artwork"];
	        this.artwork_mime = source["artwork_mime"];
	        this.artwork_b64 = source["artwork_b64"];
	        this.summary = source["summary"];
	    }
	}
	export class WriteAudioTagsInput {
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    track_total: number;
	    disc_number: number;
	    disc_total: number;
	    cover_path: string;
	    clear_artwork: boolean;
	    sort_tags: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WriteAudioTagsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.track_total = source["track_total"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	        this.cover_path = source["cover_path"];
	        this.clear_artwork = source["clear_artwork"];
	        this.sort_tags = source["sort_tags"];
	    }
	}

}

export namespace splice {
	
	export class AlbumMetadata {
	    album: string;
	    album_artist: string;
	    artist: string;
	    year: string;
	    genre: string;
	    artwork_path?: string;
	    total_tracks?: number;
	
	    static createFrom(source: any = {}) {
	        return new AlbumMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.artist = source["artist"];
	        this.year = source["year"];
	        this.genre = source["genre"];
	        this.artwork_path = source["artwork_path"];
	        this.total_tracks = source["total_tracks"];
	    }
	}
	export class MasterProbe {
	    duration_ms: number;
	    sample_rate: number;
	    channels: number;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new MasterProbe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration_ms = source["duration_ms"];
	        this.sample_rate = source["sample_rate"];
	        this.channels = source["channels"];
	        this.summary = source["summary"];
	    }
	}
	export class PeakBin {
	    min: number;
	    max: number;
	
	    static createFrom(source: any = {}) {
	        return new PeakBin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min = source["min"];
	        this.max = source["max"];
	    }
	}
	export class Track {
	    title: string;
	    duration_ms: number;
	    start_ms?: number;
	    artist?: string;
	    track_number?: number;
	    duration?: string;
	    album?: string;
	    album_artist?: string;
	    genre?: string;
	    year?: string;
	    disc_number?: number;
	    disc_total?: number;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.duration_ms = source["duration_ms"];
	        this.start_ms = source["start_ms"];
	        this.artist = source["artist"];
	        this.track_number = source["track_number"];
	        this.duration = source["duration"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	    }
	}
	export class Project {
	    master_path: string;
	    output_dir: string;
	    album: AlbumMetadata;
	    tracks: Track[];
	    master_duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.master_path = source["master_path"];
	        this.output_dir = source["output_dir"];
	        this.album = this.convertValues(source["album"], AlbumMetadata);
	        this.tracks = this.convertValues(source["tracks"], Track);
	        this.master_duration_ms = source["master_duration_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WaveformPeaks {
	    bins: PeakBin[];
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveformPeaks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bins = this.convertValues(source["bins"], PeakBin);
	        this.duration_ms = source["duration_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	    "youtube-mode": boolean;
	    "yt-dlp-path": string;
	    "youtube-save-folder": string;
	
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
	        this["youtube-mode"] = source["youtube-mode"];
	        this["yt-dlp-path"] = source["yt-dlp-path"];
	        this["youtube-save-folder"] = source["youtube-save-folder"];
	    }
	}

}

export namespace youtube {
	
	export class DownloadMeta {
	    num: number;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    disc_number: number;
	    track_total: number;
	    art_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.disc_number = source["disc_number"];
	        this.track_total = source["track_total"];
	        this.art_url = source["art_url"];
	    }
	}

}

