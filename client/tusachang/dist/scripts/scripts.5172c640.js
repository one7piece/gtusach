"use strict";var urlPrefix="http://110.175.4.240",showAlert=function(a,b,c){a.show(a.alert().parent(angular.element(document.querySelector("#popupContainer"))).clickOutsideToClose(!0).title(b).textContent(c).ariaLabel("Alert Dialog").ok("OK"))};angular.module("tusachangApp",["ngMaterial","ngAnimate","ngCookies","ngResource","ngRoute","ngSanitize","ngTouch","ui.router"]).filter("trusted",["$sce",function(a){return function(b){return a.trustAsResourceUrl(b)}}]).config(["$mdThemingProvider","$mdIconProvider","$routeProvider","$stateProvider","$httpProvider",function(a,b,c,d,e){b.iconSet("core","./images/core-icons.svg",24).icon("person","./images/person.svg").icon("login","./images/logout.svg"),a.theme("default").primaryPalette("blue"),d.state("bookshelf",{url:"/bookshelf",templateUrl:"views/bookshelf.html",controller:"BookshelfCtrl as bookshelf"}).state("createBook",{url:"/createBook",templateUrl:"views/createbook.html",controller:"CreateBookCtrl as creator"})}]),angular.module("tusachangApp").factory("myHttpResponseInterceptor",["$q","$rootScope",function(a,b){return{responseError:function(c){return 401==c.status&&b.$emit("authentication","sessionExpired"),a.reject(c)}}}]),angular.module("tusachangApp").directive("ngEnter",function(){return function(a,b,c){b.bind("keydown keypress",function(b){console.log("ngEnter - key:"+b.which),13===b.which&&(a.$apply(function(){a.$eval(c.ngEnter)}),b.preventDefault())})}}),angular.module("tusachangApp").service("BookService",["$rootScope","$http","$timeout",function(a,b,c){var d=this;d.sites=[],d.books=[],d.abortingBooks=[],d.cacheProps={},d.cacheProps.sortRange=[{desc:"Book Title (A..Z)",name:"title",order:"asc"},{desc:"Book Title (Z..A)",name:"title",order:"dsc"},{desc:"Update Time (Oldest first)",name:"time",order:"asc"},{desc:"Update Time (Newest first)",name:"time",order:"dsc"},{desc:"Owner (A..Z)",name:"owner",order:"asc"},{desc:"Owner (Z..A)",name:"owner",order:"dsc"}],d.cacheProps.sortBy=d.cacheProps.sortRange[3],d.cacheProps.showOnlyMyBooks=!0,d.subscribe=function(b,c){var d=a.$on("bookService",function(a,b){c(b.name,b.data)});b.$on("$destroy",d)},d.notify=function(b,c){a.$emit("bookService",{name:b,data:c})},d.loadSites=function(a){d.sites.length>0&&a(!0,d.sites),console.log("loading sites...");var c={headers:{"Content-Type":"application/json"}};b.get(urlPrefix+"/api/sites",c).success(function(b,c){console.log("BookService.loadSites() - success response, status:"+c,b),200===c&&b?(d.sites=b,a&&a(!0,d.sites)):a&&a(!1,"Failed to load sites, status="+c)}).error(function(b,c){console.log("BookService.loadSites() - error response, status:"+c),a&&a(!1,"Failed to load sites, status="+c)})},d.loadBooks=function(a){console.log("loading all books...");var c={headers:{"Content-Type":"application/json"}};b.get(urlPrefix+"/api/books/0",c).then(function(b){200==b.status&&b.data&&(d.books=b.data,a?a(!0,d.books):d.notify("books",d.books))},function(b){a&&a(!1,"Failed to load books, status="+b.status)})},d.loadBook=function(a,c){console.log("download book: "+a);var e={headers:{"Content-Type":"application/json"}};b.get(urlPrefix+"/api/books/0",e).success(function(a,b){console.log("BookService.loadBooks() - success response, status:"+b,a),200===b&&a?(d.books=a,c(!0,d.books)):c(!1,"Failed to load books, status="+b)}).error(function(a,b){console.log("BookService.loadBooks() - error response, status:"+b),c(!1,"Failed to load books, status="+b)})},d.updateBook=function(e,f,g){console.log(f+" book: "+e);var h={headers:{"Content-Type":"application/json"}};b.post(urlPrefix+"/api/book/"+a.logonUser.sessionId+"/"+f,e,h).success(function(a,b){console.log("BookService.updateBook() - success response, status:"+b,a),200===b&&a?("abort"==f&&(d.abortingBooks.push(e),c(function(){for(var a=0;a<d.abortingBooks.length;a++)if(d.abortingBooks[a].id==e.id){d.abortingBooks.splice(a,1);break}},2e4)),g(!0,a)):g(!1,"Failed to "+f+" book, status="+b)}).error(function(a,b){console.log("BookService.updateBook() - error response, status:"+b,a),g(!1,"Failed to "+f+" book, status="+b)})},d.refreshInterval=1e4,d.systemInfo={},d.loadSystemInfo=function(){var a={headers:{"Content-Type":"application/json"}};b.get(urlPrefix+"/api/systeminfo",a).success(function(a,b){if(200===b&&a){var e=d.systemInfo.bookLastUpdateTime!=a.bookLastUpdateTime;d.systemInfo=a,e&&d.loadBooks(null)}else console.log("Failed to load systemInfo, status="+b);c(d.loadSystemInfo,d.refreshInterval)}).error(function(a,b){console.log("Failed to load systemInfo, status="+b),c(d.loadSystemInfo,d.refreshInterval)})},d.loadSystemInfo(),c(d.loadSystemInfo,d.refreshInterval)}]),angular.module("tusachangApp").service("LoginService",["$rootScope","$cookies","$http",function(a,b,c){var d=this;a.logonUser={name:"",role:"",sessionId:""};var e=b.get("tusachangApp.sessionid");void 0!=e&&null!=e&&""!=e&&(a.logonUser.sessionId=e),a.isLogin=!1,console.log("login-service - isLogin:"+a.isLogin+", sessionId:"+a.logonUser.sessionId),d.doValidateSession=function(b){var e={headers:{"Content-Type":"application/json"}};c.get(urlPrefix+"/api/user/"+a.logonUser.sessionId,e).success(function(c,e){console.log("doValidateSession, status:"+e+", ",c),200==e&&c&&c.sessionId&&""!=c.sessionId&&(a.logonUser=c,a.isLogin=!0),d.updateCookie(a.isLogin),a.$emit("authentication"),b&&b(200==e,e)}).error(function(c,e){d.updateCookie(!1),a.$emit("authentication"),b&&b(!1,e)})},""!=a.logonUser.sessionId&&d.doValidateSession(null),d.doLogin=function(b,e,f){console.log("doLogin - urlPrefix:"+urlPrefix);var g={name:b,password:e},h={headers:{"Content-Type":"application/json"}};a.isLogin=!1,a.logonUser.sessionId="",a.logonUser.name="",c.post(urlPrefix+"/api/login",g,h).success(function(b,c){var e="";200==c&&b&&b.sessionId&&""!=b.sessionId?(console.log("success login, status:"+c+", data:",b),a.logonUser=b,a.isLogin=!0,a.$emit("authentication")):(e="Error logging in to server: "+c,b&&(e=b.status),console.log("error login, status:"+c+", msg:"+e)),f(a.isLogin,e),d.updateCookie(a.isLogin)}).error(function(a,b){console.log("error login, status:"+b+", data:",a);var c="Error logging in to server: "+b;f(!1,c),d.updateCookie(!1)})},d.doLogout=function(b){d.updateCookie(!1);var e={headers:{"Content-Type":"application/json"}};c.post(urlPrefix+"/api/logout/"+a.logonUser.sessionId,e).success(function(a,c){console.log("success logout, status:"+c),b&&(a?b(200==c,a.status):b(!1,"Failed to logout, status="+c))}).error(function(a,c){console.log("error logout, status:"+c),b&&b(!1,"Failed to logout, status="+c)}),a.$emit("authentication","logout")},d.updateCookie=function(c){if(c){var d=new Date,e=36e5;d.setDate(d.getTime()+e),b.put("tusachangApp.sessionid",a.logonUser.sessionId,{expires:d})}else b.remove("tusachangApp.sessionid"),a.logonUser.sessionId="",a.logonUser.name="",a.isLogin=!1}}]),angular.module("tusachangApp").controller("MainCtrl",["$rootScope","$state","$scope","LoginService",function(a,b,c,d){var e=this;e.username="",e.password="";var f={Android:function(){return navigator.userAgent.match(/Android/i)},BlackBerry:function(){return navigator.userAgent.match(/BlackBerry/i)},iOS:function(){return navigator.userAgent.match(/iPhone|iPad|iPod/i)},Opera:function(){return navigator.userAgent.match(/Opera Mini/i)},Windows:function(){return navigator.userAgent.match(/IEMobile/i)},any:function(){return f.Android()||f.BlackBerry()||f.iOS()||f.Opera()||f.Windows()}};console.log("isMobile: "+f.any()),e.desktopMode=1!=f.any,a.desktopMode=e.desktopMode,b.go("bookshelf"),c.$watch(function(a){return e.desktopMode},function(b){console.log("desktopMode changed, newValue: "+e.desktopMode),a.desktopMode=e.desktopMode}),a.$on("authentication",function(c,e){console.log("received event: "+c+", data:"+e+", isLogin:"+a.isLogin);var f="";"sessionExpired"==e&&(d.updateCookie(!1),f="Your session has expired!"),1!=a.isLogin&&"bookshelf"!=b.current.name&&b.go("bookshelf",{message:f})}),e.signInOut=function(){1!=a.isLogin?(console.log("sign in with "+e.username+"/"+e.password),d.doLogin(e.username,e.password,function(a,b){a||console.log("Failed to login! "+b)})):d.doLogout(function(a,b){})}}]),angular.module("tusachangApp").controller("LoginCtrl",["$rootScope","$scope","$stateParams","$interval","LoginService",function(a,b,c,d,e){var f=this;console.log("LoginCtrl ..."),b.username="",b.password="",b.statusMessage=c.message,b.statusType=""!=b.statusMessage?"error":"info",b.rebooting=c.rebooting===!0,b.rebooting&&(b.monitorTimer=d(function(){console.log("validating session with server..."),e.doValidateSession(function(a,b){(200==b||401==b)&&f.stopTimer()})},1e4)),b.$on("$destroy",function(){f.stopTimer()}),b.doLogin=function(){b.statusType="info",b.statusMessage="Authenticating with server...",e.doLogin(b.username,b.password,function(c,d){b.statusType=a.isLogin?"info":"error",b.statusMessage=d})},f.stopTimer=function(){b.rebooting=!1,b.monitorTimer&&(d.cancel(b.monitorTimer),b.monitorTimer=null)}}]),angular.module("tusachangApp").controller("BookshelfCtrl",["$rootScope","$state","$scope","$mdDialog","$filter","BookService",function(a,b,c,d,e,f){console.log("bookshelfCtrl creating...");var g=this;g.books=[],g.loading=!1,g.statusMessage="",g.statusType="info",g.sortRange=f.cacheProps.sortRange,g.sortBy=f.cacheProps.sortBy,g.showOnlyMyBooks=f.cacheProps.showOnlyMyBooks,g.sort=function(){g.sortBy&&(f.cacheProps.sortBy=g.sortBy,console.log("sort() - "+g.sortBy.desc),g.books.sort(function(a,b){var c=0;return"time"==g.sortBy.name?(a.lastUpdatedTime<b.lastUpdatedTime?c=-1:a.lastUpdatedTime>b.lastUpdatedTime&&(c=1),"dsc"==g.sortBy.order&&(c*=-1)):("title"==g.sortBy.name?c=a.title.toLowerCase().localeCompare(b.title.toLowerCase()):"owner"==g.sortBy.name&&(c=a.createdBy.toLowerCase().localeCompare(b.createdBy.toLowerCase())),0==c?a.lastUpdatedTime<b.lastUpdatedTime?c=1:a.lastUpdatedTime>b.lastUpdatedTime&&(c=-1):"dsc"==g.sortBy.order&&(c*=-1)),c}))},g.updateDisplay=function(){console.log("updateDisplay: showOnlyMyBooks:"+g.showOnlyMyBooks),void 0!=g.showOnlyMyBooks&&(f.cacheProps.showOnlyMyBooks=g.showOnlyMyBooks,g.processBooks(f.books))},g.processBooks=function(b){g.books=[];for(var c=0;c<b.length;c++){var d=b[c],f="WORKING"!=d.status&&(1==g.showOnlyMyBooks&&d.createdBy==a.logonUser.name||0==g.showOnlyMyBooks);if(f){var h=d.currentPageNo+"/"+d.maxNumPages;e("date")(new Date(d.lastUpdatedTime),"dd-MM-yyyy HH:mm");d.titleSummary=d.title+"("+h+")",g.books.push(d)}}g.statusType="info",g.statusMessage=g.books.length+" books loaded",g.sort()},g.load=function(a){a?(g.statusMessage="Loading books from server...",g.loading=!0,f.loadBooks(function(a,b){g.loading=!1,a?g.processBooks(b):(g.statusMessage="Error loading books: "+b,g.statusType="error")})):g.processBooks(f.books)},g.select=function(b){console.log("select book: ",b),a.desktopMode&&(a.selectedBook=b)},g.getDownloadLink=function(a){return urlPrefix+"/downloadBook/"+a.title+".epub?bookId="+a.id},g.canUpdate=function(b,c){return"download"==c?!0:"WORKING"==b.status||!a.isLogin||a.logonUser.name!=b.createdBy&&-1==a.logonUser.role.indexOf("admin")?!1:!0},g.update=function(a,b){if("delete"==b){var c=d.confirm().title("Confirm Delete").textContent("Do you really want to delete: "+a.title).ariaLabel("Lucky day").ok("Yes").cancel("Cancel");d.show(c).then(function(){g.doUpdate(a,b)},function(){})}else g.doUpdate(a,b)},g.doUpdate=function(a,b){c.loading=!0,f.updateBook(a,b,function(d,e){if(c.loading=!1,d){if(g.statusMessage="","delete"==b||"resume"==b)for(var f=0;f<g.books.length;f++)a.id==g.books[f].id&&(console.log("remove local cache of book index:"+f+", "+a.title),g.books.splice(f,1))}else g.statusMessage=e,g.statusType="error"})},a.$on("reload",function(){"bookshelf"==b.current.name&&g.load(!0)}),f.subscribe(c,function(a,b){console.log("bookshelf: received BookService event: "+a),console.log(b),"books"==a&&g.processBooks(b)}),g.processBooks(f.books)}]),angular.module("tusachangApp").controller("BookDetailCtrl",["$rootScope","$state","$scope","$mdBottomSheet","BookService",function(a,b,c,d,e){var f=this;c.loading=!1,c.statusMessage="",c.statusType="info",c.book=a.selectedBook,console.log("book detail: "+c.book),c.book.statusSummary=c.book.status,"ERROR"==c.book.status&&(c.book.statusSummary+=" ("+c.book.errorMsg+")"),c.download=function(){window.location=urlPrefix+"/downloadBook/"+c.book.title+".epub?bookId="+c.book.id},c.canUpdate=function(b){return"download"==b?!0:"WORKING"==book.status||!a.isLogin||a.logonUser.name!=c.book.createdBy&&-1==a.logonUser.role.indexOf("admin")?!1:!0},c.update=function(a){c.loading=!0,e.updateBook(c.book,a,function(a,b){c.loading=!1,a?(f.statusMessage=b,f.statusType="error"):f.statusMessage=""})}}]),angular.module("tusachangApp").controller("CreateBookCtrl",["$rootScope","$state","$scope","$mdDialog","$interval","BookService",function(a,b,c,d,e,f){var g=this;g.loading=!1,g.sites=[],g.books=[],g.firstChapterURL="",g.title="",g.author="",g.numPages=0,g.statusMessage="",g.statusType="info",g.create=function(){if(""==g.firstChapterURL)return void showAlert(d,"Invalid Data","Must specify the book First Chapter URL!");if(""==g.title)return void showAlert(d,"Invalid Data","Must specify the book Title!");g.statusMessage="Creating book...",g.statusType="info";var a={startPageUrl:g.firstChapterURL,title:g.title,author:g.author,maxNumPages:g.numPages};g.firstChapterURL="",g.title="",g.author="",g.numPages=0,f.updateBook(a,"create",function(a,b){a?g.statusMessage="":(g.statusMessage=b,g.statusType="error")})},g.canAbort=function(a){for(var b=0;b<f.abortingBooks.length;b++)if(f.abortingBooks[b].id==a.id)return!1;return!0},g.abort=function(a){f.updateBook(a,"abort",function(a,b){a?g.statusMessage="":(g.statusMessage=b,g.statusType="error")})},g.processBooks=function(a){g.books=[];for(var b=0;b<a.length;b++)"WORKING"==a[b].status&&(g.books.push(a[b]),a[b].details=a[b].status+" ("+a[b].currentPageNo+"/"+a[b].maxNumPages+")")},g.load=function(){g.loading=!0,f.loadSites(function(a,b){a&&(g.sites=b)}),f.loadBooks(function(a,b){g.loading=!1,a?g.processBooks(b):g.books=[]})},f.subscribe(c,function(a,b){console.log("createbook: received BookService event: "+a),console.log(b),"books"==a&&g.processBooks(b)}),g.load()}]),angular.module("tusachangApp").run(["$templateCache",function(a){a.put("views/bookdetail.html",'<md-bottom-sheet class="md-has-header"> <md-subheader>{{book.title}}</md-subheader> <md-progress-linear ng-if="loading" md-mode="indeterminate"></md-progress-linear> <span ng-if="statusMessage.length > 0" class="{{statusType}}Message-bookshelf">{{bookshelf.statusMessage}}</span> <div layout="column" layout-align="center center"> <div flex layout="row" class="row-bookdetail"> <md-input-container class="field-bookdetail-first"> <label>Title</label> <input ng-model="book.title"> </md-input-container> <md-input-container class="field-bookdetail-second"> <label>Author</label> <input ng-model="book.author"> </md-input-container> </div> <div flex layout="row" class="row-bookdetail"> <md-input-container class="field-bookdetail-first"> <label>Last Update Time</label> <input readonly ng-model="book.lastUpdatedTime"> </md-input-container> <md-input-container class="field-bookdetail-remaining"> <label>Status</label> <input readonly ng-model="book.statusSummary"> </md-input-container> </div> <div flex layout="row" class="row-bookdetail"> <md-input-container class="field-bookdetail-first"> <label>Current Page No</label> <input readonly ng-model="book.currentPageNo"> </md-input-container> <md-input-container class="field-bookdetail-second"> <label>Max Num Pages</label> <input ng-model="book.maxNumPages"> </md-input-container> </div> <md-input-container class="row-bookdetail"> <label>Start Page URL</label> <input readonly ng-model="book.startPageUrl"> </md-input-container> <md-input-container class="row-bookdetail"> <label>Current Page URL</label> <input ng-model="book.currentPageUrl"> </md-input-container> <div> <md-button ng-disabled="book.status == \'WORKING\' || !book.epubCreated" ng-click="download()" class="md-raised md-primary">Download</md-button> <md-button ng-disabled="!canUpdate(\'delete\')" ng-click="update(\'delete\')" class="md-raised md-warn">Delete</md-button> <md-button ng-disabled="!canUpdate(\'resume\')" ng-click="update(\'resume\')" class="md-raised md-primary">Resume</md-button> <md-button ng-disabled="!canUpdate(\'update\')" ng-click="update(\'update\')" class="md-raised md-primary">Save Changes</md-button> </div> </div> </md-bottom-sheet>'),a.put("views/bookshelf.html",'<div class="panel-bookshelf" flex> <md-progress-linear ng-if="bookshelf.loading" md-mode="indeterminate"></md-progress-linear> <div layout="column" style="margin-left: 20px; margin-top:5px" layout-align="start"> <div> <span flex ng-if="bookshelf.statusMessage.length > 0" class="{{bookshelf.statusType}}Message-bookshelf">{{bookshelf.statusMessage}}</span> </div> <div layout="row" layout-align="start end"> <div> <md-input-container style="padding:0px;margin:0px"> <md-select ng-model="bookshelf.sortBy" ng-model-options="{trackBy: \'$value.desc\'}" style="width:100%" ng-change="bookshelf.sort()"> <md-option ng-repeat="option in bookshelf.sortRange" ng-value="{{option}}">{{option.desc}}</md-option> </md-select> </md-input-container> </div> <div style="margin-left:10px"> <md-checkbox ng-model="bookshelf.showOnlyMyBooks" ng-change="bookshelf.updateDisplay()" aria-label="checkbox1">My Books Only</md-checkbox> </div> </div> </div> <md-content> <md-list> <md-list-item class="md-2-line" ng-repeat="book in bookshelf.books"> <md-button ng-href="{{bookshelf.getDownloadLink(book)}}" style="min-width:58px;height:48px" aria-label="{{book.title}}"> <img src="./images/book.png"> </md-button> <div class="md-list-item-text" layout="column"> <p style="color:blue">{{book.title}} ({{book.currentPageNo}}/{{book.maxNumPages}})</p> <p>{{book.createdBy}}, {{book.lastUpdatedTime | date:\'dd/MM/yyyy HH:mm:ss\'}}, {{book.status}}</p> </div> <md-button style="min-width:40px" ng-disabled="!bookshelf.canUpdate(book, \'resume\')" ng-click="bookshelf.update(book, \'resume\')"> <md-tooltip>Resume</md-tooltip> <md-icon md-svg-icon="core:resume" style="width:36px; {{bookshelf.canUpdate(book, \'resume\') ? \'color: blue\' : \'color: grey\'}}"></md-icon> </md-button> <md-button style="min-width:40px" ng-disabled="!bookshelf.canUpdate(book, \'delete\')" ng-click="bookshelf.update(book, \'delete\')"> <md-tooltip>Delete</md-tooltip> <md-icon md-svg-icon="core:delete" style="{{bookshelf.canUpdate(book, \'delete\') ? \'color: red\' : \'color: grey\'}}"></md-icon> </md-button> </md-list-item> </md-list> </md-content> </div>'),a.put("views/createbook.html",'<div class="panel-createbook" layout="column" flex> <md-progress-linear ng-if="creator.loading" md-mode="indeterminate"></md-progress-linear> <span ng-if="creator.statusMessage.length > 0" class="{{creator.statusType}}Message-bookshelf">{{creator.statusMessage}}</span> <div>Supported Sites:</div> <div layout="row" layout-wrap style="margin-left:10px"> <div ng-repeat="site in creator.sites"> <a href="{{site}}" target="_blank" style="margin-right:10px; margin-top:10px; margin-bottom:10px">{{site}}</a> </div> </div> <md-divider style="margin-top:10px"></md-divider> <md-grid-list flex md-cols="1" md-cols-sm="1" md-cols-gt-sm="2" md-cols-md="2" md-cols-gt-md="2" md-cols-lg="2" md-cols-gt-lg="2" md-gutter="12px" md-row-height="50px"> <md-grid-tile> <md-input-container style="width:400px"> <label>First Chapter URL</label> <input ng-model="creator.firstChapterURL"> </md-input-container> </md-grid-tile> <md-grid-tile> <md-input-container style="width:400px"> <label>Title</label> <input ng-model="creator.title"> </md-input-container> </md-grid-tile> <md-grid-tile> <md-input-container style="width:400px"> <label>Author</label> <input ng-model="creator.author"> </md-input-container> </md-grid-tile> <md-grid-tile> <md-input-container style="width:400px"> <label>Num Pages</label> <input ng-model="creator.numPages"> </md-input-container> </md-grid-tile> </md-grid-list> <div style="margin-top:20px"> <md-button ng-click="creator.create()" class="md-raised md-primary">Create</md-button> </div> <span ng-if="statusMessage.length > 0" class="{{statusType}}-message">{{statusMessage}}</span> <md-divider></md-divider> <md-content> <md-list> <md-subheader class="md-no-sticky">Pending Books</md-subheader> <md-list-item class="md-2-line" ng-repeat="book in creator.books"> <md-button ng-disabled="!creator.canAbort(book)" ng-click="creator.abort(book)" aria-label="Abort"> <md-tooltip>Abort</md-tooltip> <md-icon md-svg-src="./images/abort.svg" style="color: red"></md-icon> </md-button> <div class="md-list-item-text" layout="column"> <p style="color:blue">{{book.title}} ({{book.currentPageNo}}/{{book.maxNumPages}})</p> <p>{{book.status}} - {{book.lastUpdatedTime | date:\'dd/MM/yyyy HH:mm:ss\'}}</p> </div> </md-list-item> </md-list> </md-content> </div>')}]);