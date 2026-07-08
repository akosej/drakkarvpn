export namespace main {
	
	export class Profile {
	    id: string;
	    name: string;
	    privateKey: string;
	    publicKey: string;
	    address: string;
	    dns: string;
	    endpoint: string;
	    allowedIPs: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.privateKey = source["privateKey"];
	        this.publicKey = source["publicKey"];
	        this.address = source["address"];
	        this.dns = source["dns"];
	        this.endpoint = source["endpoint"];
	        this.allowedIPs = source["allowedIPs"];
	    }
	}

}

