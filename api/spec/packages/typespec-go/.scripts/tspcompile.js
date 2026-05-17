// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
import { exec, execSync } from 'child_process';
import { existsSync, opendirSync, unlinkSync, readFileSync, writeFileSync } from 'fs';
import { semaphore } from '../../../.scripts/semaphore.js';

// limit to 8 concurrent builds
const sem = semaphore(8);

const pkgRoot = execSync('git rev-parse --show-toplevel').toString().trim() + '/packages/typespec-go/';

const httpSpecs = pkgRoot + 'node_modules/@typespec/http-specs/specs/';
const azureHttpSpecs = pkgRoot + 'node_modules/@azure-tools/azure-http-specs/specs/';

const compiler = pkgRoot + 'node_modules/@typespec/compiler/cmd/tsp.js';

// the format is as follows
// 'moduleName': [ 'input', 'emitter option 1', 'emitter option N...' ]
// if no .tsp file is specified in input, it's assumed to be main.tsp
const httpSpecsGroup = {
  'apikeygroup': ['authentication/api-key'],
  'customgroup': ['authentication/http/custom'],
  'oauth2group': ['authentication/oauth2'],
  'unionauthgroup': ['authentication/union'],
  'bytesgroup': ['encode/bytes'],
  'datetimegroup': ['encode/datetime', 'slice-elements-byval=true'],
  'durationgroup': ['encode/duration'],
  'numericgroup': ['encode/numeric'],
  'basicparamsgroup': ['parameters/basic'],
  'bodyoptionalgroup': ['parameters/body-optionality'],
  'collectionfmtgroup': ['parameters/collection-format'],
  'pathgroup': ['parameters/path'],
  'querygroup': ['parameters/query'],
  'spreadgroup': ['parameters/spread'],
  'contentneggroup': ['payload/content-negotiation'],
  'jmergepatchgroup': ['payload/json-merge-patch'],
  'mediatypegroup': ['payload/media-type'],
  'multipartgroup': ['payload/multipart'],
  'pageablegroup': ['payload/pageable'], // missing support for continuation tokens: https://github.com/Azure/autorest.go/issues/1494
  'xmlgroup': ['payload/xml', 'slice-elements-byval=true'],
  //'statuscoderangegroup': ['response/status-code-range'], // TODO: https://github.com/Azure/autorest.go/issues/1606
  //'routesgroup': ['routes'], // TODO: https://github.com/Azure/autorest.go/issues/1730
  'jsongroup': ['serialization/encoded-name/json'],
  'noendpointgroup': ['server/endpoint/not-defined'],
  'multiplegroup': ['server/path/multiple'],
  'singlegroup': ['server/path/single'],
  'unversionedgroup': ['server/versions/not-versioned'],
  'versionedgroup': ['server/versions/versioned'],
  'condreqgroup': ['special-headers/conditional-request'],
  //'repeatabilitygroup': ['special-headers/repeatability'],   // requires union support
  'specialwordsgroup': ['special-words'],
  //'jsonlgroup': ['streaming/jsonl'], // TODO: https://github.com/Azure/autorest.go/issues/1594
  'arraygroup': ['type/array', 'slice-elements-byval=true'],
  'dictionarygroup': ['type/dictionary'],
  'extensiblegroup': ['type/enum/extensible'],
  'fixedgroup': ['type/enum/fixed'],
  'emptygroup': ['type/model/empty', 'single-client=true'],
  'enumdiscgroup': ['type/model/inheritance/enum-discriminator'],
  //'nesteddiscgroup': ['type/model/inheritance/nested-discriminator'], // not a real scenario
  'nodiscgroup': ['type/model/inheritance/not-discriminated'],
  'recursivegroup': ['type/model/inheritance/recursive', 'slice-elements-byval=true'],
  'singlediscgroup': ['type/model/inheritance/single-discriminator'],
  'usagegroup': ['type/model/usage'],
  'visibilitygroup': ['type/model/visibility'],
  //'addlpropsgroup': ['type/property/additional-properties'], // requires union support (remove hand-written client when done)
  'nullablegroup': ['type/property/nullable'],
  'optionalitygroup': ['type/property/optionality', 'slice-elements-byval=true'], // missing support for plain time https://github.com/Azure/autorest.go/issues/1732
  'valuetypesgroup': ['type/property/value-types', 'slice-elements-byval=true'],
  'scalargroup': ['type/scalar', 'slice-elements-byval=true'],
  //'uniondiscriminatedgroup': ['type/union/discriminated'], // requires union support
  //'uniongroup': ['type/union'], // requires union support
  //'addedgroup': ['versioning/added'], // requires union support
  'madeoptionalgroup': ['versioning/madeOptional'],
  //'removedgroup': ['versioning/removed'], // requires union support
  //'renamedfromgroup': ['versioning/renamedFrom'], // requires union support
  'rettypechangedfromgroup': ['versioning/returnTypeChangedFrom'],
  'typechangedfromgroup': ['versioning/typeChangedFrom'],
};

const azureHttpSpecsGroup = {
  'accessgroup': ['azure/client-generator-core/access'],
  //'alternatetypegroup': ['azure/client-generator-core/alternate-type'],
  'defaultvaluegroup': ['azure/client-generator-core/client-default-value'],
  'emptystringgroup': ['azure/client-generator-core/deserialize-empty-string-as-null'],
  'flattengroup': ['azure/client-generator-core/flatten-property'],
  'nextlinkverbgroup': ['azure/client-generator-core/next-link-verb', 'slice-elements-byval=true'],
  'coreusagegroup': ['azure/client-generator-core/usage'],
  'overridegroup': ['azure/client-generator-core/override/client.tsp'],
  'hierarchygroup': ['azure/client-generator-core/hierarchy-building'],
  'clientinitdefaultgroup': ['azure/client-generator-core/client-initialization/default'],
  'clientinitindividuallygroup': ['azure/client-generator-core/client-initialization/individually'],
  'clientinitindividuallyparentgroup': ['azure/client-generator-core/client-initialization/individuallyParent'],
  'apiversionheadergroup' : ['azure/client-generator-core/api-version/header/client.tsp'],
  'apiversionpathgroup' : ['azure/client-generator-core/api-version/path/client.tsp'],
  'apiversionquerygroup' : ['azure/client-generator-core/api-version/query/client.tsp'],
  'basicgroup': ['azure/core/basic'],
  'lrorpcgroup': ['azure/core/lro/rpc'],
  'lrostdgroup': ['azure/core/lro/standard'],
  'azurepagegroup': ['azure/core/page/client.tsp'], // requires paging with re-injection support
  'corescalargroup': ['azure/core/scalar'],
  'coremodelgroup': ['azure/core/model'],
  'coreclientlocationmovemethodparametertoclientgroup': ['azure/client-generator-core/client-location/move-method-parameter-to-client'],
  'coreclientlocationmoveexistingsubclientgroup': ['azure/client-generator-core/client-location/move-to-existing-sub-client'],
  'coreclientlocationmovenewsubclientgroup': ['azure/client-generator-core/client-location/move-to-new-sub-client'],
  'coreclientlocationmoverootclientgroup': ['azure/client-generator-core/client-location/move-to-root-client'],
  // 'coredeserializegroup': ['azure/client-generator-core/deserialize-empty-string-as-null'],
  'traitsgroup': ['azure/core/traits'], // requires union support
  'encodedurationgroup': ['azure/encode/duration'],
  'examplebasicgroup': ['azure/example/basic'],
  'pageablegroup': ['azure/payload/pageable'],
  'commonpropsgroup': ['azure/resource-manager/common-properties'],
  'resources': ['azure/resource-manager/resources', 'factory-gather-all-params=false'],
  'nonresourcegroup' : ['azure/resource-manager/non-resource'],
  'templatesgroup' : ['azure/resource-manager/operation-templates'],
  'largeheadergroup' : ['azure/resource-manager/large-header'],
  'methodsubscriptionidgroup' : ['/azure/resource-manager/method-subscription-id/client.tsp'],
  'armmultipleservicegroup' : ['/azure/resource-manager/multi-service/client.tsp'],
  'armmultisharedmodelsgroup' : ['/azure/resource-manager/multi-service-shared-models/client.tsp'],
  'xmsclientreqidgroup': ['azure/special-headers/client-request-id'],
  'previewversiongroup': ['azure/versioning/previewVersion', 'api-version=2024-12-01-preview'],
  'previewversiongroupspecificversion': ['azure/versioning/previewVersion', 'api-version=2024-06-01'],
  'naminggroup': ['client/naming'],
  'enumconflictgroup': ['client/naming/enum-conflict/client.tsp'],
  'defaultgroup': ['client/structure/default/client.tsp'],
  // disable operation group related tests, will re-enable after new TCGC version released
  //'multiclientgroup': ['client/structure/multi-client/client.tsp'],
  //'renamedopgroup': ['client/structure/renamed-operation/client.tsp'],
  //'clientopgroup': ['client/structure/client-operation-group/client.tsp'],
  //'twoopgroup': ['client/structure/two-operation-group/client.tsp'],
  'clientnamespacegroup': ['client/namespace'],
  'overloadgroup': ['client/overload/client.tsp'],
  'srvdrivenoldgroup': ['resiliency/srv-driven/old.tsp'],
  'srvdrivennewgroup': ['resiliency/srv-driven'],
  'multipleservicesgroup' : ['/service/multiple-services'],
  'multiservicegroup' : ['/service/multi-service'],
};

// default to using the locally built emitter
let emitter = pkgRoot;
const args = process.argv.slice(2);
var filter = undefined;
const switches = [];
for (var i = 0 ; i < args.length; i += 1) {
  const filterArg = args[i].match(/--filter=(?<filter>\w+)/);
  if (filterArg) {
    filter = filterArg.groups['filter'];
    continue;
  }

  switch (args[i]) {
    case '--verbose':
      switches.push('--verbose');
      break;
    case '--emitter-installed':
      // the emitter has been installed so use that one instead
      emitter = '@azure-tools/typespec-go';
      break;
    default:
      break;
  }
}

if (filter !== undefined) {
  console.log("Using filter: " + filter)
}

function should_generate(name) {
  if (filter !== undefined) {
    const re = new RegExp(filter);
    return re.test(name)
  }
  return true
}

const armcodesigning = pkgRoot + 'test/tsp/CodeSigning.Management';
generate('armcodesigning', armcodesigning, 'test/local/armcodesigning', [`examples-directory=${armcodesigning}/examples`, 'generate-samples=true']);

const armapicenter = pkgRoot +  'test/tsp/ApiCenter.Management';
generate('armapicenter', armapicenter, 'test/local/armapicenter', [`examples-directory=${armapicenter}/examples`, 'generate-samples=true']);

const armcognitiveservices = pkgRoot +  'test/tsp/CognitiveServices.Management';
generate('armcognitiveservices', armcognitiveservices, 'test/local/armcognitiveservices', [`examples-directory=${armcognitiveservices}/examples`, 'generate-samples=true']);

const armlargeinstance = pkgRoot + 'test/tsp/AzureLargeInstance.Management';
generate('armlargeinstance', armlargeinstance, 'test/local/armlargeinstance', ['stutter=AzureLargeInstance', `examples-directory=${armlargeinstance}/examples`, 'generate-samples=true']);

const armdatabasewatcher = pkgRoot + 'test/tsp/DatabaseWatcher.Management';
generate('armdatabasewatcher', armdatabasewatcher, 'test/local/armdatabasewatcher', ['fix-const-stuttering=false', `examples-directory=${armdatabasewatcher}/examples`, 'generate-samples=true']);

const armloadtestservice = pkgRoot + 'test/tsp/LoadTestService.Management';
generate('armloadtestservice', armloadtestservice, 'test/local/armloadtestservice', [`examples-directory=${armloadtestservice}/examples`, 'generate-samples=true', 'factory-gather-all-params=false']);

const armdevopsinfrastructure = pkgRoot + 'test/tsp/Microsoft.DevOpsInfrastructure';
generate('armdevopsinfrastructure', armdevopsinfrastructure, 'test/local/armdevopsinfrastructure', [`examples-directory=${armdevopsinfrastructure}/examples`, 'generate-samples=true']);

const armrandom = pkgRoot + 'test/tsp/Random.Management';
generate('armrandom', armrandom, 'test/local/armrandom');

const armcommunitymanagement = pkgRoot + 'test/tsp/Community.Management';
generate('armcommunitymanagement', armcommunitymanagement, 'test/local/armcommunitymanagement', [`examples-directory=${armcommunitymanagement}/examples`, 'generate-samples=true']);

const armmongocluster = pkgRoot + 'test/tsp/MongoCluster.Management';
generate('armmongocluster', armmongocluster, 'test/local/armmongocluster', [`examples-directory=${armmongocluster}/examples`, 'generate-samples=true']);

const armcontainerorchestratorruntime = pkgRoot + 'test/tsp/KubernetesRuntime.Management';
generate('armcontainerorchestratorruntime', armcontainerorchestratorruntime, 'test/local/armcontainerorchestratorruntime', [`examples-directory=${armcontainerorchestratorruntime}/examples`, 'generate-samples=true']);

const azmodelsonly = pkgRoot + 'test/tsp/ModelsOnlyWithBaseTypes';
generate('azmodelsonly', azmodelsonly, 'test/local/azmodelsonly');

const azkeys = pkgRoot + 'test/tsp/KeyVault.Keys/client.tsp';
generate('azkeys', azkeys, 'test/local/azkeys', ['single-client=true', 'omit-constructors=true']);

const azblob = pkgRoot + 'test/tsp/Microsoft.BlobStorage/client.tsp';
generate('azblob', azblob, 'test/local/azblob', ['generate-fakes=false', 'omit-constructors=true', 'inject-spans=false']);

const armtest = pkgRoot + 'test/tsp/Test.Management';
generate('armtest/v2', armtest, 'test/local/armtest', [`examples-directory=${armtest}/examples`, 'generate-samples=true']);

const internalpager = pkgRoot + 'test/tsp/Internal.Pager';
generate('internalpager', internalpager, 'test/local/internalpager', ['generate-fakes=false']);

const armoracledatabase = pkgRoot + 'test/tsp/Oracle.Database.Management';
generate('armoracledatabase/v2', armoracledatabase, 'test/local/armoracledatabase', [`examples-directory=${armoracledatabase}/examples`, 'generate-samples=true']);

const armhealthbot = pkgRoot + 'test/tsp/Healthbot.Management';
generate('armhealthbot', armhealthbot, 'test/local/armhealthbot', [`examples-directory=${armhealthbot}/examples`, 'generate-samples=true']);

const armhardwaresecuritymodules = pkgRoot + 'test/tsp/HardwareSecurityModules.Management';
generate('armhardwaresecuritymodules', armhardwaresecuritymodules, 'test/local/armhardwaresecuritymodules', [`examples-directory=${armhardwaresecuritymodules}/examples`, 'generate-samples=true']);

const armcomputeschedule = pkgRoot + 'test/tsp/ComputeSchedule.Management';
generate('armcomputeschedule', armcomputeschedule, 'test/local/armcomputeschedule', [`examples-directory=${armcomputeschedule}/examples`, 'generate-samples=true']);

const armbillingbenefits = pkgRoot + 'test/tsp/BillingBenefits.Management';
generate('armbillingbenefits', armbillingbenefits, 'test/local/armbillingbenefits', [`examples-directory=${armbillingbenefits}/examples`, 'generate-samples=true']);

const nooptionalbody = pkgRoot + 'test/tsp/NoOptionalBody';
generate('nooptionalbody', nooptionalbody, 'test/local/nooptionalbody', ['generate-fakes=false', 'go-generate=after_generate.go', 'no-optional-body=true']);

const rawjson = pkgRoot + 'test/tsp/RawJson';
generate('rawjson', rawjson, 'test/local/rawjson/subpkg', ['containing-module=rawjson/v2', 'rawjson-as-bytes=true']);

const azregressions = pkgRoot + 'test/tsp/Regressions';
generate('azregressions', azregressions, 'test/local/azregressions');

const azclientoption = pkgRoot + 'test/tsp/ClientOption';
generate('azclientoption', azclientoption, 'test/local/azclientoption');

const armpageablelros = pkgRoot + 'test/tsp/PageableLROs';
generate('armpageablelros', armpageablelros, 'test/local/armpageablelros');

loopSpec(httpSpecsGroup, httpSpecs, 'test/http-specs')
loopSpec(azureHttpSpecsGroup, azureHttpSpecs, 'test/azure-http-specs')

function loopSpec(group, root, prefix) {
  for (const module in group) {
    const values = group[module];
    let perTestOptions;
    if (values.length > 1) {
      perTestOptions = values.slice(1);
    }
    // keep the output directory structure similar to the cadl input directory.
    // remove the last dir from the input path as we'll use the module name instead.
    // if the input specifies a .tsp file, remove that first.
    let outDir = values[0];
    if (outDir.lastIndexOf('.tsp') > -1) {
      outDir = outDir.substring(0, outDir.lastIndexOf('/'));
    }
    outDir = outDir.substring(0, outDir.lastIndexOf('/'));
    generate(module, root + values[0], `${prefix}/${outDir}/` + module, perTestOptions);
  }
}

function generate(moduleName, input, outputDir, perTestOptions) {
  if (!should_generate(moduleName)) {
    return
  }
  if (perTestOptions === undefined) {
    perTestOptions = [];
  }

  const fullOutputDir = pkgRoot + outputDir;

  // check for containing-module and swap out module for it as needed
  let outputKind = `module=${moduleName}`;
  for (const perTestOption of perTestOptions) {
    if (perTestOption.match(/containing\-module/)) {
      outputKind = perTestOption;
      break;
    }
  }

  // these options can't be changed per test
  const fixedOptions = [
    outputKind,
    `emitter-output-dir=${fullOutputDir}`,
    'file-prefix=zz_',
  ];

  // these options _can_ be changed per test
  // TODO: disabled examples by default https://github.com/Azure/autorest.go/issues/1441
  const defaultOptions = [
    'generate-fakes=true',
    'inject-spans=true',
    'head-as-boolean=true',
    'fix-const-stuttering=true',
  ];

  let allOptions = fixedOptions;

  // merge in any per-test options.
  // if a per-test option overlaps with a default option, use the per-test one.
  for (const perTestOption of perTestOptions) {
    // perTestOption === 'option=value', grab the option name
    const optionName = perTestOption.match(/^([a-zA-Z0-9_-]+)/)[0];
    const index = defaultOptions.findIndex((value, index, obj) => {
      return value.startsWith(optionName);
    });
    if (index > -1) {
      // found a match, replace the default option with the per-test one
      defaultOptions[index] = perTestOption;
    } else {
      allOptions.push(perTestOption);
    }
  }

  allOptions = allOptions.concat(defaultOptions);

  sem.take(function() {
    // if a tsp file isn't specified, first check
    // for a client.tsp file. if that doesn't exist
    // then fall back to main.tsp.
    if (input.lastIndexOf('.tsp') === -1) {
      if (existsSync(input + '/client.tsp')) {
        input += '/client.tsp';
      } else {
        input += '/main.tsp';
      }
    }
    console.log('generating ' + input);
    const options = [];
    for (const option of allOptions) {
      options.push(`--option="@azure-tools/typespec-go.${option}"`);
    }
    if (switches.includes('--debugger')) {
      options.push(`--option="@azure-tools/typespec-go.debugger=true"`);
    }
    const command = `node ${compiler} compile ${input} --emit=${emitter} ${options.join(' ')}`;
    if (switches.includes('--verbose')) {
      console.log(command);
    }
    exec(command, function(error, stdout, stderr) {
      try {
        // print any output or error from the tsp compile command
        logResult(error, stdout, stderr);
        // format on success
        if (error === null) {
          // Force emitter version to a constant in _metadata.json to avoid unnecessary version drift in committed files
          const metadataPath = `${fullOutputDir}/testdata/_metadata.json`;
          if (existsSync(metadataPath)) {
            const metadata = JSON.parse(readFileSync(metadataPath, 'utf8'));
            metadata.emitterVersion = '0.0.0';
            writeFileSync(metadataPath, JSON.stringify(metadata, null, 2) + '\n');
          }
        } else {
          // delete files on error so it's easy to spot codegen failures
          cleanGeneratedFiles(fullOutputDir);
        }
      } catch (err) {
        console.error('An error occurred:');
        if (err.message) {
          console.error('Message:', err.message);
        }
        if (err.stack) {
          console.error('Stack:', err.stack);
        }
        if (err.output) {
          console.error('Output:', err.output.toString());
        }
      } finally {
        sem.leave();
      }
    });
  });
}

function cleanGeneratedFiles(outputDir) {
  if (!existsSync(outputDir)) {
      return;
  }
  const dir = opendirSync(outputDir);
  while (true) {
      const dirEnt = dir.readSync()
      if (dirEnt === null) {
          break;
      }
      // preserve the version.go file so we can test the v2+ major version scenario
      if (dirEnt.isFile() && dirEnt.name.startsWith('zz_') && dirEnt.name !== 'zz_version.go') {
          unlinkSync(dir.path + '/' + dirEnt.name);
      }
  }
  dir.close();
  cleanGeneratedFiles(outputDir + '/fake');
}

function logResult(error, stdout, stderr) {
  if (stdout !== '') {
    console.log('stdout: ' + stdout);
  }
  // typespec compiler prints compiler progress to stderr
  // but it's not an error, so we use console.log
  // to print it out.
  if (stderr !== '') {
    console.log('stderr: ' + stderr);
  }
  if (error !== null) {
    console.error('\x1b[91m%s\x1b[0m', 'exec error: ' + error);
  }
}
