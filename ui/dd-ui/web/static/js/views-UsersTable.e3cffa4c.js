(window["webpackJsonp"]=window["webpackJsonp"]||[]).push([["views-UsersTable"],{"2a7f":function(t,e,i){"use strict";i.d(e,"a",(function(){return n}));var s=i("71d9"),a=i("80d2"),n=Object(a["j"])("v-toolbar__title"),r=Object(a["j"])("v-toolbar__items");s["a"]},"36a7":function(t,e,i){},"5e23":function(t,e,i){},"71d9":function(t,e,i){"use strict";var s=i("3835"),a=i("5530"),n=(i("a9e3"),i("0481"),i("5e23"),i("8dd9")),r=i("adda"),o=i("80d2"),c=i("d9bd");e["a"]=n["a"].extend({name:"v-toolbar",props:{absolute:Boolean,bottom:Boolean,collapse:Boolean,dense:Boolean,extended:Boolean,extensionHeight:{default:48,type:[Number,String]},flat:Boolean,floating:Boolean,prominent:Boolean,short:Boolean,src:{type:[String,Object],default:""},tag:{type:String,default:"header"}},data:function(){return{isExtended:!1}},computed:{computedHeight:function(){var t=this.computedContentHeight;if(!this.isExtended)return t;var e=parseInt(this.extensionHeight);return this.isCollapsed?t:t+(isNaN(e)?0:e)},computedContentHeight:function(){return this.height?parseInt(this.height):this.isProminent&&this.dense?96:this.isProminent&&this.short?112:this.isProminent?128:this.dense?48:this.short||this.$vuetify.breakpoint.smAndDown?56:64},classes:function(){return Object(a["a"])(Object(a["a"])({},n["a"].options.computed.classes.call(this)),{},{"v-toolbar":!0,"v-toolbar--absolute":this.absolute,"v-toolbar--bottom":this.bottom,"v-toolbar--collapse":this.collapse,"v-toolbar--collapsed":this.isCollapsed,"v-toolbar--dense":this.dense,"v-toolbar--extended":this.isExtended,"v-toolbar--flat":this.flat,"v-toolbar--floating":this.floating,"v-toolbar--prominent":this.isProminent})},isCollapsed:function(){return this.collapse},isProminent:function(){return this.prominent},styles:function(){return Object(a["a"])(Object(a["a"])({},this.measurableStyles),{},{height:Object(o["h"])(this.computedHeight)})}},created:function(){var t=this,e=[["app","<v-app-bar app>"],["manual-scroll",'<v-app-bar :value="false">'],["clipped-left","<v-app-bar clipped-left>"],["clipped-right","<v-app-bar clipped-right>"],["inverted-scroll","<v-app-bar inverted-scroll>"],["scroll-off-screen","<v-app-bar scroll-off-screen>"],["scroll-target","<v-app-bar scroll-target>"],["scroll-threshold","<v-app-bar scroll-threshold>"],["card","<v-app-bar flat>"]];e.forEach((function(e){var i=Object(s["a"])(e,2),a=i[0],n=i[1];t.$attrs.hasOwnProperty(a)&&Object(c["a"])(a,n,t)}))},methods:{genBackground:function(){var t={height:Object(o["h"])(this.computedHeight),src:this.src},e=this.$scopedSlots.img?this.$scopedSlots.img({props:t}):this.$createElement(r["a"],{props:t});return this.$createElement("div",{staticClass:"v-toolbar__image"},[e])},genContent:function(){return this.$createElement("div",{staticClass:"v-toolbar__content",style:{height:Object(o["h"])(this.computedContentHeight)}},Object(o["t"])(this))},genExtension:function(){return this.$createElement("div",{staticClass:"v-toolbar__extension",style:{height:Object(o["h"])(this.extensionHeight)}},Object(o["t"])(this,"extension"))}},render:function(t){this.isExtended=this.extended||!!this.$scopedSlots.extension;var e=[this.genContent()],i=this.setBackgroundColor(this.color,{class:this.classes,style:this.styles,on:this.$listeners});return this.isExtended&&e.push(this.genExtension()),(this.src||this.$scopedSlots.img)&&e.unshift(this.genBackground()),t(this.tag,i,e)}})},"8a79":function(t,e,i){"use strict";var s=i("23e7"),a=i("06cf").f,n=i("50c4"),r=i("5a34"),o=i("1d80"),c=i("ab13"),l=i("c430"),d="".endsWith,h=Math.min,u=c("endsWith"),m=!l&&!u&&!!function(){var t=a(String.prototype,"endsWith");return t&&!t.writable}();s({target:"String",proto:!0,forced:!m&&!u},{endsWith:function(t){var e=String(o(this));r(t);var i=arguments.length>1?arguments[1]:void 0,s=n(e.length),a=void 0===i?s:h(n(i),s),c=String(t);return d?d.call(e,c,a):e.slice(a-c.length,a)===c}})},"8efc":function(t,e,i){},a523:function(t,e,i){"use strict";i("4de4"),i("b64b"),i("2ca0"),i("99af"),i("20f6"),i("4b85"),i("498a"),i("a15b");var s=i("2b0e");function a(t){return s["a"].extend({name:"v-".concat(t),functional:!0,props:{id:String,tag:{type:String,default:"div"}},render:function(e,i){var s=i.props,a=i.data,n=i.children;a.staticClass="".concat(t," ").concat(a.staticClass||"").trim();var r=a.attrs;if(r){a.attrs={};var o=Object.keys(r).filter((function(t){if("slot"===t)return!1;var e=r[t];return t.startsWith("data-")?(a.attrs[t]=e,!1):e||"string"===typeof e}));o.length&&(a.staticClass+=" ".concat(o.join(" ")))}return s.id&&(a.domProps=a.domProps||{},a.domProps.id=s.id),e(s.tag,a,n)}})}var n=i("d9f7");e["a"]=a("container").extend({name:"v-container",functional:!0,props:{id:String,tag:{type:String,default:"div"},fluid:{type:Boolean,default:!1}},render:function(t,e){var i,s=e.props,a=e.data,r=e.children,o=a.attrs;return o&&(a.attrs={},i=Object.keys(o).filter((function(t){if("slot"===t)return!1;var e=o[t];return t.startsWith("data-")?(a.attrs[t]=e,!1):e||"string"===typeof e}))),s.id&&(a.domProps=a.domProps||{},a.domProps.id=s.id),t(s.tag,Object(n["a"])(a,{staticClass:"container",class:Array({"container--fluid":s.fluid}).concat(i||[])}),r)}})},adda:function(t,e,i){"use strict";var s=i("53ca"),a=(i("a9e3"),i("a15b"),i("8a79"),i("2ca0"),i("8efc"),i("90a2")),n=(i("36a7"),i("24b2")),r=i("58df"),o=Object(r["a"])(n["a"]).extend({name:"v-responsive",props:{aspectRatio:[String,Number],contentClass:String},computed:{computedAspectRatio:function(){return Number(this.aspectRatio)},aspectStyle:function(){return this.computedAspectRatio?{paddingBottom:1/this.computedAspectRatio*100+"%"}:void 0},__cachedSizer:function(){return this.aspectStyle?this.$createElement("div",{style:this.aspectStyle,staticClass:"v-responsive__sizer"}):[]}},methods:{genContent:function(){return this.$createElement("div",{staticClass:"v-responsive__content",class:this.contentClass},this.$slots.default)}},render:function(t){return t("div",{staticClass:"v-responsive",style:this.measurableStyles,on:this.$listeners},[this.__cachedSizer,this.genContent()])}}),c=o,l=i("7560"),d=i("d9f7"),h=i("d9bd"),u="undefined"!==typeof window&&"IntersectionObserver"in window;e["a"]=Object(r["a"])(c,l["a"]).extend({name:"v-img",directives:{intersect:a["a"]},props:{alt:String,contain:Boolean,eager:Boolean,gradient:String,lazySrc:String,options:{type:Object,default:function(){return{root:void 0,rootMargin:void 0,threshold:void 0}}},position:{type:String,default:"center center"},sizes:String,src:{type:[String,Object],default:""},srcset:String,transition:{type:[Boolean,String],default:"fade-transition"}},data:function(){return{currentSrc:"",image:null,isLoading:!0,calculatedAspectRatio:void 0,naturalWidth:void 0,hasError:!1}},computed:{computedAspectRatio:function(){return Number(this.normalisedSrc.aspect||this.calculatedAspectRatio)},normalisedSrc:function(){return this.src&&"object"===Object(s["a"])(this.src)?{src:this.src.src,srcset:this.srcset||this.src.srcset,lazySrc:this.lazySrc||this.src.lazySrc,aspect:Number(this.aspectRatio||this.src.aspect)}:{src:this.src,srcset:this.srcset,lazySrc:this.lazySrc,aspect:Number(this.aspectRatio||0)}},__cachedImage:function(){if(!(this.normalisedSrc.src||this.normalisedSrc.lazySrc||this.gradient))return[];var t=[],e=this.isLoading?this.normalisedSrc.lazySrc:this.currentSrc;this.gradient&&t.push("linear-gradient(".concat(this.gradient,")")),e&&t.push('url("'.concat(e,'")'));var i=this.$createElement("div",{staticClass:"v-image__image",class:{"v-image__image--preload":this.isLoading,"v-image__image--contain":this.contain,"v-image__image--cover":!this.contain},style:{backgroundImage:t.join(", "),backgroundPosition:this.position},key:+this.isLoading});return this.transition?this.$createElement("transition",{attrs:{name:this.transition,mode:"in-out"}},[i]):i}},watch:{src:function(){this.isLoading?this.loadImage():this.init(void 0,void 0,!0)},"$vuetify.breakpoint.width":"getSrc"},mounted:function(){this.init()},methods:{init:function(t,e,i){if(!u||i||this.eager){if(this.normalisedSrc.lazySrc){var s=new Image;s.src=this.normalisedSrc.lazySrc,this.pollForSize(s,null)}this.normalisedSrc.src&&this.loadImage()}},onLoad:function(){this.getSrc(),this.isLoading=!1,this.$emit("load",this.src),this.image&&(this.normalisedSrc.src.endsWith(".svg")||this.normalisedSrc.src.startsWith("data:image/svg+xml"))&&(this.image.naturalHeight&&this.image.naturalWidth?(this.naturalWidth=this.image.naturalWidth,this.calculatedAspectRatio=this.image.naturalWidth/this.image.naturalHeight):this.calculatedAspectRatio=1)},onError:function(){this.hasError=!0,this.$emit("error",this.src)},getSrc:function(){this.image&&(this.currentSrc=this.image.currentSrc||this.image.src)},loadImage:function(){var t=this,e=new Image;this.image=e,e.onload=function(){e.decode?e.decode().catch((function(e){Object(h["c"])("Failed to decode image, trying to render anyway\n\n"+"src: ".concat(t.normalisedSrc.src)+(e.message?"\nOriginal error: ".concat(e.message):""),t)})).then(t.onLoad):t.onLoad()},e.onerror=this.onError,this.hasError=!1,this.sizes&&(e.sizes=this.sizes),this.normalisedSrc.srcset&&(e.srcset=this.normalisedSrc.srcset),e.src=this.normalisedSrc.src,this.$emit("loadstart",this.normalisedSrc.src),this.aspectRatio||this.pollForSize(e),this.getSrc()},pollForSize:function(t){var e=this,i=arguments.length>1&&void 0!==arguments[1]?arguments[1]:100,s=function s(){var a=t.naturalHeight,n=t.naturalWidth;a||n?(e.naturalWidth=n,e.calculatedAspectRatio=n/a):t.complete||!e.isLoading||e.hasError||null==i||setTimeout(s,i)};s()},genContent:function(){var t=c.options.methods.genContent.call(this);return this.naturalWidth&&this._b(t.data,"div",{style:{width:"".concat(this.naturalWidth,"px")}}),t},__genPlaceholder:function(){if(this.$slots.placeholder){var t=this.isLoading?[this.$createElement("div",{staticClass:"v-image__placeholder"},this.$slots.placeholder)]:[];return this.transition?this.$createElement("transition",{props:{appear:!0,name:this.transition}},t):t[0]}}},render:function(t){var e=c.options.render.call(this,t),i=Object(d["a"])(e.data,{staticClass:"v-image",attrs:{"aria-label":this.alt,role:this.alt?"img":void 0},class:this.themeClasses,directives:u?[{name:"intersect",modifiers:{once:!0},value:{handler:this.init,options:this.options}}]:void 0});return e.children=[this.__cachedSizer,this.__cachedImage,this.__genPlaceholder(),this.genContent()],t(e.tag,i,e.children)}})},c1b6:function(t,e,i){"use strict";i.r(e);var s=function(){var t=this,e=t.$createElement,i=t._self._c||e;return i("v-data-table",{staticClass:"elevation-1",attrs:{headers:t.headers,items:t.items},scopedSlots:t._u([{key:"top",fn:function(){return[i("v-toolbar",{attrs:{flat:""}},[i("v-toolbar-title",[t._v("Users")]),i("v-divider",{staticClass:"mx-4",attrs:{inset:"",vertical:""}}),i("v-spacer"),i("v-dialog",{attrs:{"max-width":"500px"},scopedSlots:t._u([{key:"activator",fn:function(e){var s=e.on,a=e.attrs;return[i("v-btn",t._g(t._b({staticClass:"mb-2",attrs:{color:"primary",dark:""}},"v-btn",a,!1),s),[t._v(" New User ")])]}}]),model:{value:t.dialog,callback:function(e){t.dialog=e},expression:"dialog"}},[i("v-card",[i("v-card-title",[i("span",{staticClass:"text-h5"},[t._v(t._s(t.formTitle))])]),i("v-card-text",[i("v-container",[i("v-row",[i("v-col",{attrs:{cols:"12",md:"12"}},[i("v-text-field",{attrs:{label:"Full name"},model:{value:t.editedItem.fullname,callback:function(e){t.$set(t.editedItem,"fullname",e)},expression:"editedItem.fullname"}})],1),i("v-col",{attrs:{cols:"12",md:"12"}},[i("v-text-field",{attrs:{label:"E-mail"},model:{value:t.editedItem.email,callback:function(e){t.$set(t.editedItem,"email",e)},expression:"editedItem.email"}})],1)],1)],1)],1),i("v-card-actions",[i("v-spacer"),i("v-btn",{attrs:{color:"blue darken-1",text:""},on:{click:t.close}},[t._v(" Cancel ")]),i("v-btn",{attrs:{color:"blue darken-1",text:""},on:{click:t.save}},[t._v(" Save ")])],1)],1)],1),i("v-dialog",{attrs:{"max-width":"500px"},model:{value:t.dialogDelete,callback:function(e){t.dialogDelete=e},expression:"dialogDelete"}},[i("v-card",[i("v-card-title",{staticClass:"text-h5"},[t._v(" Are you sure you want to delete this item? ")]),i("v-card-actions",[i("v-spacer"),i("v-btn",{attrs:{color:"blue darken-1",text:""},on:{click:t.closeDelete}},[t._v(" Cancel ")]),i("v-btn",{attrs:{color:"blue darken-1",text:""},on:{click:t.deleteItemConfirm}},[t._v(" OK ")]),i("v-spacer")],1)],1)],1)],1)]},proxy:!0},{key:"item.actions",fn:function(e){var s=e.item;return[i("v-icon",{staticClass:"mr-2",attrs:{small:""},on:{click:function(e){return t.editItem(s)}}},[t._v(" mdi-pencil ")]),i("v-icon",{attrs:{small:""},on:{click:function(e){return t.deleteItem(s)}}},[t._v(" mdi-delete ")])]}}])})},a=[],n=(i("a434"),i("c5fa")),r={name:"UsersTablesView",data:function(){return{snackbar:!1,snackbarMessage:"",snackbarType:"warning",dialog:!1,dialogDelete:!1,search:"",loading:!1,headers:[{text:"ID",align:"start",filterable:!1,value:"id",width:20},{text:"Full name",value:"fullname",width:150},{text:"E-mail",value:"email"},{text:"Actions",value:"actions",width:1,sortable:!1}],items:[],editedIndex:-1,editedItem:{fullname:"",email:""},defaultItem:{fullname:"",email:""}}},computed:{formTitle:function(){return-1===this.editedIndex?"New Item":"Edit Item"}},created:function(){var t=this;this.loading=!0,n["a"].get("user").then((function(e){t.items=e.data,t.loading=!1})).catch((function(t){console.log("ERROR response: "+JSON.stringify(t))}))},methods:{initialize:function(){},editItem:function(t){this.editedIndex=this.items.indexOf(t),this.editedItem=Object.assign({},t),this.dialog=!0},deleteItem:function(t){this.editedIndex=this.items.indexOf(t),this.editedItem=Object.assign({},t),this.dialogDelete=!0},deleteItemConfirm:function(){var t=this;this.items.splice(this.editedIndex,1),this.closeDelete(),console.log("item: "+JSON.stringify(this.editedItem)),n["a"].delete("data/users/"+this.editedItem.ID,this.editedItem).then((function(e){t.$notification.success("User successfully deleted!")})).catch((function(e){t.$notification.error("Failed to delete user!")}))},close:function(){var t=this;this.dialog=!1,this.$nextTick((function(){t.editedItem=Object.assign({},t.defaultItem),t.editedIndex=-1}))},closeDelete:function(){var t=this;this.dialogDelete=!1,this.$nextTick((function(){t.editedItem=Object.assign({},t.defaultItem),t.editedIndex=-1}))},save:function(){var t=this,e=this.editedItem;this.editedIndex>-1?(Object.assign(this.items[this.editedIndex],this.editedItem),n["a"].put("data/users",this.editedItem).then((function(e){t.$notification.success("User "+e.data.fullname+" successfully updated!")})).catch((function(i){t.$notification.error("Failed to update user!"+i+", "+JSON.stringify(e))}))):(this.items.push(this.editedItem),n["a"].post("data/users",this.editedItem).then((function(e){t.$notification.success("User "+e.data.fullname+" successfully added!")})).catch((function(i){t.$notification.error("Failed to add user!"+i+", "+JSON.stringify(e))}))),this.close()}}},o=r,c=i("2877"),l=i("6544"),d=i.n(l),h=i("8336"),u=i("b0af"),m=i("99d9"),f=i("62ad"),p=i("a523"),g=i("8fea"),v=i("169a"),b=i("ce7e"),S=i("132d"),y=i("0fd9"),x=i("2fa4"),_=i("8654"),I=i("71d9"),C=i("2a7f"),O=Object(c["a"])(o,s,a,!1,null,null,null);e["default"]=O.exports;d()(O,{VBtn:h["a"],VCard:u["a"],VCardActions:m["a"],VCardText:m["b"],VCardTitle:m["c"],VCol:f["a"],VContainer:p["a"],VDataTable:g["a"],VDialog:v["a"],VDivider:b["a"],VIcon:S["a"],VRow:y["a"],VSpacer:x["a"],VTextField:_["a"],VToolbar:I["a"],VToolbarTitle:C["a"]})}}]);
//# sourceMappingURL=views-UsersTable.e3cffa4c.js.map