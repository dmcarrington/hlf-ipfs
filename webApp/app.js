// Run with: DEBUG=webApp:* npm start

const createError = require('http-errors');
const express = require('express');
const path = require('path');
const cookieParser = require('cookie-parser');
const logger = require('morgan');
const passport     = require('passport'),
    exphbs = require('express-handlebars'),
    bodyParser   = require('body-parser'),
    methodOverride = require('method-override'),
    session = require('express-session'),
    LdapStrategy = require('passport-ldapauth');

// If this is running in a Docker container, the url
// must be the name of the container, else localhost.
var OPTS = {
  passReqToCallback: true,
  server: {
    url: 'ldap://ldap-server:389',
    bindDN: 'cn=admin,dc=example,dc=com',
    bindCredentials: 'adminpw',
    searchBase: 'dc=example,dc=com',
    searchFilter: '(cn={{username}})'
  }
};
const fileUpload = require('express-fileupload');
var app = express();

app.use(fileUpload());

passport.use(new LdapStrategy(OPTS));

// view engine setup
app.set('views', path.join(__dirname, 'views'));
app.set('view engine', 'hbs');

app.use(logger('dev'));
app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(cookieParser());
app.use(express.static(path.join(__dirname, 'public')));

app.use(bodyParser.json());
app.use(bodyParser.urlencoded({extended: false}));
app.use(methodOverride('X-HTTP-Method-Override'));
app.use(session({secret: 'supernova', saveUninitialized: true, resave: true}));
app.use(passport.initialize());
app.use(passport.session());

// Session-persisted message middleware
app.use(function(req, res, next){
  var err = req.session.error,
      msg = req.session.notice,
      success = req.session.success;

  delete req.session.error;
  delete req.session.success;
  delete req.session.notice;

  if (err) res.locals.error = err;
  if (msg) res.locals.notice = msg;
  if (success) res.locals.success = success;

  next();
});

// Configure express to use handlebars templates
var hbs = exphbs.create({
  defaultLayout: 'main', //we will be creating this layout shortly
});
app.engine('handlebars', hbs.engine);
app.set('view engine', 'handlebars');

//===============ROUTES=================
//displays our homepage
app.get('/', async function(req, res){
  
  let username = undefined;
  let userEnrolled = false;
  let userOriginatedTransfers = undefined;
  let userRecipientTransfers = undefined;
  if(req.session.user) {
    username = req.session.user.cn;
    userEnrolled = true;
    var fabricClient = require('./config/FabricClient');
    await fabricClient.initCredentialStores();
    await fabricClient.getCertificateAuthority();
    let user = await fabricClient.getUserContext(username, true);
    if(user) {
      // query the chaincode for transfers created by this user
      let fcn = "queryAllTransfers";
      const args = [username];
      const queryChaincode = require('./invoke.js').queryChaincode;
      let chaincodeContent = await queryChaincode(fabricClient, fcn, args);

      console.log("Setting userOriginatedTransfers to ", chaincodeContent.payload.responses[0]);
      userOriginatedTransfers = chaincodeContent.payload.responses[0];

      // Repeat the query for transfers assigned to us
      /*fcn = "queryTransfersByRecipient";
      chaincodeContent = await queryChaincode(fabricClient, fcn, args);
      console.log("Setting userRecipientTransfers to ", chaincodeContent.payload.responses[0]);
      userRecipientTransfers = chaincodeContent.payload.responses[0];*/
    }
  }
  res.render('home', {
    user: username, 
    userEnrolled: userEnrolled, 
    userOriginatedTransfers: userOriginatedTransfers, 
    userRecipientTransfers: userRecipientTransfers
  });

});

//displays our signup page
app.get('/signin', function(req, res){
  res.render('signin');
});

//sends the request through our local signup strategy, and if successful takes user to homepage, otherwise returns then to signin page

app.post('/upload-file', async function(req, res) {

    // Got our file with its content courtesy of express-fileupload
    // file.data contains the actual contents of the file
    const file = req.files.uploadFilename;
    const fileContent = file.data;

    const writeToIPFS = require('./ipfs').writeToIPFS;
    const commitHash = await writeToIPFS(fileContent);
    if(commitHash === 0) {
      res.render('home', {user: req.session.user.cn, message: "Failed to commit file to IPFS"});
    }
    else {
      console.log(req);
      const recipient = req.body.recipient;
      const msg = "File commited sucessfully: " + commitHash;
      const fcn = "createTransfer";
      const args = [req.session.user.cn, commitHash, recipient];
      var fabricClient = require('./config/FabricClient');
      await fabricClient.initCredentialStores();
      await fabricClient.getCertificateAuthority();
      await fabricClient.getUserContext(req.session.user.cn, true);
      const proposeTransaction = require('./invoke.js').proposeTransaction;
      const proposalObject = await proposeTransaction(fabricClient, fcn, args);
      if (!proposalObject.success){
        res.send({
          success: 500, 
          message: "Unable to commit your transaction"});
      }
      
      const commitTransaction = require('./invoke.js').commitTransaction;
      const committedObject = await commitTransaction(fabricClient, 
        proposalObject.payload.txId, 
        proposalObject.payload.proposalResponses, 
        proposalObject.payload.proposal);
      if (!committedObject.success){
        res.send({
          success: 500, 
          message: "Unable to commit your transaction"});
      }

      if (committedObject.payload.commitStatus == 'SUCCESS'){
        console.log("success!");
        /*const results = await blockchain.attachEventHub(clientObject.payload.client, proposalObject.payload.txIDString, 3000);
        res.send({
          success: 200, 
          message: results});*/
      }
      res.render('home', {user: req.session.user.cn, message: msg});
    }

});

app.post('/local-reg', async function(req, res) {
    const username = req.body.username;
    const password = req.body.password;
    const enrolUser = require('./enrolUser').enrolUser;
    const result = await enrolUser(username, password);
    /*const clientObj = await blockchain.getClient(username, password);
    if (clientObj.success) {
    
      const registerUserObj = await blockchain.registerUser(clientObj.payload.client, clientObj.payload.enrolledUserObj, registrantName);
      console.log(registerUserObj);*/
    if(result === 'ok') {
      res.render('signin', {message: "User enrolled successfully"})
    }
    else {
      console.log(clientObj);
      res.render('signin', {message: "Failed to enrol user with Hyperledger Fabric network. Please ensure that the \
      username and password are valid on the LDAP server"})
    }
  }
);

//logs user out of site, deleting them from the session, and returns to homepage
app.get('/logout', function(req, res){
  var name = req.session.user.cn;
  console.log("LOGGING OUT " + req.session.user.cn)
  req.logout();
  req.session.user = null;
  res.redirect('/');
  req.session.notice = "You have successfully been logged out " + name + "!";
});

app.post('/login', async function(req, res, next) {
  passport.authenticate('ldapauth', async function (err, user, info){
    if(user){
      console.log("successfully authenticated user:", user);
      req.session.secret = req.body.password;
      req.session.user = user;
      res.redirect('/');
    } else {
      res.redirect('/signin');
    }
  })(req, res, next);
});

// catch 404 and forward to error handler
app.use(function(req, res, next) {
  next(createError(404));
});

// error handler
app.use(function(err, req, res, next) {
  // set locals, only providing error in development
  res.locals.message = err.message;
  res.locals.error = req.app.get('env') === 'development' ? err : {};

  // render the error page
  res.status(err.status || 500);
  res.render('error');
});

module.exports = app;
