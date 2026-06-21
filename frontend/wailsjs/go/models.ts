export namespace main {
	
	export class CreateIconRequest {
	    inputPath: string;
	    outputPath: string;
	    radius: number;
	    zoom: number;
	    panX: number;
	    panY: number;
	    keepIntermediates: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateIconRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.radius = source["radius"];
	        this.zoom = source["zoom"];
	        this.panX = source["panX"];
	        this.panY = source["panY"];
	        this.keepIntermediates = source["keepIntermediates"];
	    }
	}
	export class CreateIconResponse {
	    icnsPath: string;
	    icoPath: string;
	    directory: string;
	    fileName: string;
	    icnsFileName: string;
	    icoFileName: string;
	    workingDir?: string;
	    cleanedUp: boolean;
	    replacedFile: boolean;
	    outputSize: number;
	    icnsSize: number;
	    icoSize: number;
	    statusMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateIconResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.icnsPath = source["icnsPath"];
	        this.icoPath = source["icoPath"];
	        this.directory = source["directory"];
	        this.fileName = source["fileName"];
	        this.icnsFileName = source["icnsFileName"];
	        this.icoFileName = source["icoFileName"];
	        this.workingDir = source["workingDir"];
	        this.cleanedUp = source["cleanedUp"];
	        this.replacedFile = source["replacedFile"];
	        this.outputSize = source["outputSize"];
	        this.icnsSize = source["icnsSize"];
	        this.icoSize = source["icoSize"];
	        this.statusMessage = source["statusMessage"];
	    }
	}
	export class ImageInfo {
	    path: string;
	    name: string;
	    directory: string;
	    defaultOutputPath: string;
	    width: number;
	    height: number;
	    sizeBytes: number;
	    previewDataURL: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.directory = source["directory"];
	        this.defaultOutputPath = source["defaultOutputPath"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.sizeBytes = source["sizeBytes"];
	        this.previewDataURL = source["previewDataURL"];
	    }
	}

}

